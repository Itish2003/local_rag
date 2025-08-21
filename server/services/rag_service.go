// =====================================================
// services.go
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github/itish2003/rag/models"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2" // <-- Import the types package
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/google/uuid"
	"google.golang.org/genai"
)

// RAGService interface defines methods for RAG operations
type RAGService interface {
	IngestNote(c context.Context, req models.IngestDataRequest) error
	QueryRAG(c context.Context, req models.QueryTextRequest) (*models.QueryRAGResponse, error)
	GetAllNotes(c context.Context) (*models.GetAllNotesResponse, error)
	EmbedTextWithOllama(ctx context.Context, textToEmbed string) ([]float32, error)
}

// ragServiceImpl holds the dependencies it needs to do its job
type ragServiceImpl struct {
	httpClient   *http.Client
	collection   chromago.Collection // Changed from pointer to interface
	geminiClient *genai.Client
}

// GetAllNotes implements RAGService to retrieve all documents from ChromaDB.
func (r *ragServiceImpl) GetAllNotes(c context.Context) (*models.GetAllNotesResponse, error) {
	log.Printf("SERVICE: Getting all notes from ChromaDB...")

	// Use the v2 API's Get method to retrieve all documents.
	results, err := r.collection.Get(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents from chromadb: %w", err)
	}

	// Extract the data using the correct accessor methods.
	ids := results.GetIDs()
	documents := results.GetDocuments()
	metadatas := results.GetMetadatas()

	// Check if the collection is empty.
	if len(ids) == 0 {
		log.Printf("SERVICE: No notes found in the collection.")
		return &models.GetAllNotesResponse{
			Count: 0,
			Notes: []models.Note{},
		}, nil
	}

	// Transform the results into the response model.
	notes := make([]models.Note, 0, len(documents))
	for i := range documents {
		var metadataMap map[string]interface{}
		if len(metadatas) > i && metadatas[i] != nil {
			// Marshal the DocumentMetadata to JSON
			jsonBytes, err := json.Marshal(metadatas[i])
			if err != nil {
				log.Printf("WARN: could not marshal metadata for document %s: %v", ids[i], err)
				// Assign an empty map or handle the error as appropriate
				metadataMap = make(map[string]interface{})
			} else {
				// Unmarshal the JSON back into a map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &metadataMap); err != nil {
					log.Printf("WARN: could not unmarshal metadata for document %s: %v", ids[i], err)
					metadataMap = make(map[string]interface{})
				}
			}
		}

		notes = append(notes, models.Note{
			ID:       string(ids[i]),
			Text:     documents[i].ContentString(),
			Metadata: metadataMap,
		})
	}

	log.Printf("SERVICE: Successfully retrieved %d notes", len(notes))
	return &models.GetAllNotesResponse{
		Count: len(notes),
		Notes: notes,
	}, nil
}

// IngestNote implements RAGService
func (r *ragServiceImpl) IngestNote(c context.Context, req models.IngestDataRequest) error {
	log.Printf("SERVICE: Ingesting note: '%s'", req.Text)

	embeddingVector, err := r.EmbedTextWithOllama(c, req.Text)
	if err != nil {
		return fmt.Errorf("could not generate embedding for note: %w", err)
	}

	// Create the proper embedding type
	embedding := embeddings.NewEmbeddingFromFloat32(embeddingVector)

	// Create metadata
	metadata := chromago.NewDocumentMetadata(
		chromago.NewStringAttribute("source", "user_input"),
	)

	// Use the proper embedding type
	err = r.collection.Add(c,
		chromago.WithIDs(chromago.DocumentID(uuid.New().String())),
		chromago.WithTexts(req.Text),
		chromago.WithEmbeddings(embedding),
		chromago.WithMetadatas(metadata),
	)
	if err != nil {
		return fmt.Errorf("failed to add record to chromadb: %w", err)
	}

	log.Printf("SERVICE: Successfully added document")
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

// retrieveDocuments queries ChromaDB for similar documents using v2 API
func (r *ragServiceImpl) retrieveDocuments(c context.Context, query string, nResults int) ([]string, error) {
	log.Printf("SERVICE-HELPER: Retrieving documents from ChromaDB using v2 API...")

	// 1. Embed the query text using Ollama
	queryEmbedding, err := r.EmbedTextWithOllama(c, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query text: %w", err)
	}

	// Create the proper embedding type for the query
	embedding := embeddings.NewEmbeddingFromFloat32(queryEmbedding)

	// 2. Use the query embedding to find similar documents in ChromaDB
	results, err := r.collection.Query(
		c,
		chromago.WithQueryEmbeddings(embedding),
		chromago.WithNResults(nResults),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chromadb: %w", err)
	}

	// Extract documents from results using v2 API methods
	var documents []string
	documentGroups := results.GetDocumentsGroups()

	if len(documentGroups) > 0 {
		for _, doc := range documentGroups[0] {
			if doc.ContentString() != "" {
				documents = append(documents, doc.ContentString())
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

	// Updated to use gemini-2.5-flash as mentioned in the search results
	resp, err := r.geminiClient.Models.GenerateContent(c, "gemini-2.5-flash", content, nil)
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
		return fmt.Sprintf("I don't have any relevant information to answer the question: %s", query)
	}
	context := "Context:\n" + strings.Join(retrievedDocs, "\n\n")
	prompt := fmt.Sprintf("Using only the provided context, answer the following question. If the context doesn't contain relevant information, say so.\n\n%s\n\nQuestion: %s\n\nAnswer:", context, query)
	return prompt
}

// EmbedTextWithOllama generates embeddings using Ollama.
func (r *ragServiceImpl) EmbedTextWithOllama(c context.Context, textToEmbed string) ([]float32, error) {
	reqBody, err := json.Marshal(models.OllamaEmbedRequest{
		Model:  "nomic-embed-text:v1.5",
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
func NewRAGService(client *http.Client, collection chromago.Collection, geminiClient *genai.Client) RAGService {
	return &ragServiceImpl{
		httpClient:   client,
		collection:   collection, // No longer a pointer
		geminiClient: geminiClient,
	}
}
