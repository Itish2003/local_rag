// =====================================================
// services.go
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github/itish2003/rag/models"
	"io"
	"log"
	"net/http"
	"strings"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/google/uuid"
	"google.golang.org/genai"
)

// RAGService interface defines methods for RAG operations
type RAGService interface {
	IngestNote(c context.Context, req models.IngestDataRequest) error
	QueryRAG(c context.Context, req models.QueryTextRequest) (*models.QueryRAGResponse, error)
}

// ragServiceImpl holds the dependencies it needs to do its job
type ragServiceImpl struct {
	httpClient   *http.Client
	collection   *chromago.Collection
	geminiClient *genai.Client
}

// IngestNote implements RAGService
func (r *ragServiceImpl) IngestNote(c context.Context, req models.IngestDataRequest) error {
	log.Printf("SERVICE: Ingesting note: '%s'", req.Text)

	embedding, err := r.embedTextWithOllama(c, req.Text)
	if err != nil {
		return fmt.Errorf("could not generate embedding for note: %w", err)
	}

	_, err = r.collection.Add(
		c,
		[]*types.Embedding{types.NewEmbeddingFromFloat32(embedding)},
		[]map[string]interface{}{{"source": "user_input"}},
		[]string{req.Text},
		[]string{uuid.New().String()},
	)
	if err != nil {
		return fmt.Errorf("failed to add record to chromadb: %w", err)
	}

	log.Printf("SERVICE: Successfully got embedding of size %d", len(embedding))
	return nil
}

// QueryRAG implements RAGService
func (r *ragServiceImpl) QueryRAG(c context.Context, req models.QueryTextRequest) (*models.QueryRAGResponse, error) {
	log.Printf("SERVICE: Querying RAG with: '%s'", req.Query)

	retrievedDocs, err := r.retrieveDocuments(c, req.Query, 3)
	if err != nil {
		return nil, err
	}

	ragPrompt := r.createRAGPrompt(req.Query, retrievedDocs)

	// Generate response from Gemini
	geminiAnswer, err := r.generateResponseWithGemini(c, ragPrompt)
	if err != nil {
		return nil, fmt.Errorf("could not generate response from gemini: %w", err)
	}

	response := &models.QueryRAGResponse{
		Answer:     geminiAnswer,
		SourceDocs: retrievedDocs,
	}
	return response, nil
}

// retrieveDocuments queries ChromaDB for similar documents
func (r *ragServiceImpl) retrieveDocuments(c context.Context, query string, nResults int) ([]string, error) {
	log.Printf("SERVICE-HELPER: Retrieving documents from ChromaDB...")

	results, err := r.collection.Query(
		c,
		[]string{query},
		int32(nResults),
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chromadb: %w", err)
	}

	// Extract documents from results
	var documents []string
	if len(results.Documents) > 0 {
		for _, doc := range results.Documents[0] {
			if doc != "" {
				documents = append(documents, doc)
			}
		}
	}

	log.Printf("SERVICE-HELPER: Retrieved %d documents", len(documents))
	return documents, nil
}

// generateResponseWithGemini generates a response using Gemini API
func (r *ragServiceImpl) generateResponseWithGemini(c context.Context, prompt string) (string, error) {
	log.Printf("SERVICE-HELPER: Sending prompt to Gemini...")

	// Use the correct method from the API
	parts := []*genai.Part{
		{Text: prompt},
	}
	content := []*genai.Content{
		{Parts: parts},
	}

	resp, err := r.geminiClient.Models.GenerateContent(c, "gemini-1.5-flash", content, nil)
	if err != nil {
		return "", fmt.Errorf("gemini api call failed: %w", err)
	}

	// Extract the text from the response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "I'm sorry, I couldn't generate a response based on the provided notes.", nil
	}

	// Concatenate all text parts
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText.WriteString(part.Text)
		}
	}
	return responseText.String(), nil
}

// createRAGPrompt creates a prompt with context for the LLM
func (r *ragServiceImpl) createRAGPrompt(query string, retrievedDocs []string) string {
	if len(retrievedDocs) == 0 {
		return query
	}
	context := "Context:\n" + strings.Join(retrievedDocs, "\n\n")
	prompt := fmt.Sprintf("Using only the provided context, answer the following question.\n\n%s\n\nQuestion: %s\n\nAnswer:", context, query)
	return prompt
}

// embedTextWithOllama generates embeddings using Ollama
func (r *ragServiceImpl) embedTextWithOllama(c context.Context, textToEmbed string) ([]float32, error) {
	reqBody, err := json.Marshal(models.OllamaEmbedRequest{
		Model:  "nomic-embed-text",
		Prompt: textToEmbed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(c, http.MethodPost, "http://localhost:11434/api/embeddings", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama embedding api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp models.OllamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}
	return ollamaResp.Embedding, nil
}

// NewRAGService creates a new RAG service instance
func NewRAGService(client *http.Client, collection *chromago.Collection, geminiClient *genai.Client) RAGService {
	return &ragServiceImpl{
		httpClient:   client,
		collection:   collection,
		geminiClient: geminiClient,
	}
}
