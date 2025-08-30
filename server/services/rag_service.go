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
	"sync"

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
	GetTotalChunks(c context.Context) (int, error)
}

// ragServiceImpl holds the dependencies it needs to do its job
type ragServiceImpl struct {
	httpClient   *http.Client
	collection   chromago.Collection // Changed from pointer to interface
	geminiClient *genai.Client
	FileActions  *FileActions
	chatSessions map[string]*genai.Chat
	mu           sync.Mutex
}

// GetTotalChunks counts all the document chunks in the collection.
func (r *ragServiceImpl) GetTotalChunks(c context.Context) (int, error) {
	count, err := r.collection.Count(c)
	if err != nil {
		return 0, fmt.Errorf("failed to count items in collection: %w", err)
	}
	return int(count), nil
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
	log.Printf("SERVICE: Querying RAG with: '%s' (SessionID: '%s')", req.Query, req.SessionID)

	// Lock the mutex to safely access the sessions map.
	r.mu.Lock()
	defer r.mu.Unlock()

	var session *genai.Chat
	sessionID := req.SessionID

	// If a session ID is provided, try to find the existing session.
	if sessionID != "" {
		session = r.chatSessions[sessionID]
	}

	// If no session ID was provided OR the session was not found (e.g., server restarted),
	// create a new one.
	if session == nil {
		log.Println("SERVICE: No active session found. Creating a new one.")
		var err error
		session, err = r.geminiClient.Chats.Create(c, "gemini-2.5-flash", &genai.GenerateContentConfig{
			Tools: GetFileActionTools(),
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("could not start new chat session: %w", err)
		}
		// Generate a new unique ID for the session and store it.
		sessionID = uuid.New().String()
		r.chatSessions[sessionID] = session
	}

	retrievedDocs, err := r.retrieveDocuments(c, req.Query, 3)
	if err != nil {
		return nil, err
	}

	ragPrompt := r.createRAGPrompt(req.Query, retrievedDocs)

	// Generate response from Gemini
	geminiAnswer, err := r.generateResponseWithGemini(c, session, ragPrompt)
	if err != nil {
		return nil, fmt.Errorf("could not generate response from gemini: %w", err)
	}

	response := &models.QueryRAGResponse{
		Answer:     geminiAnswer,
		SourceDocs: retrievedDocs,
		SessionID:  sessionID,
	}
	return response, nil
}

// retrieveDocuments queries ChromaDB for similar documents using v2 API
func (r *ragServiceImpl) retrieveDocuments(c context.Context, query string, nResults int) ([]models.SourceDocument, error) {
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

	var documents []models.SourceDocument
	documentGroups := results.GetDocumentsGroups()
	metadataGroups := results.GetMetadatasGroups()

	if len(documentGroups) > 0 {
		for i, doc := range documentGroups[0] {
			if doc.ContentString() != "" {
				metadata := metadataGroups[0][i]
				var metadataMap map[string]interface{}

				// THIS IS THE KEY: The DocumentMetadata struct does not have a public GetValues() method.
				// The correct way to convert it to a map is to marshal it to JSON and then unmarshal it.
				if metadata != nil {
					jsonBytes, err := json.Marshal(metadata)
					if err != nil {
						log.Printf("WARN: could not marshal metadata for document: %v", err)
						metadataMap = make(map[string]interface{}) // Use empty map on error
					} else {
						if err := json.Unmarshal(jsonBytes, &metadataMap); err != nil {
							log.Printf("WARN: could not unmarshal metadata for document: %v", err)
							metadataMap = make(map[string]interface{}) // Use empty map on error
						}
					}
				}

				sourceDoc := models.SourceDocument{
					Text:     doc.ContentString(),
					Metadata: metadataMap,
				}
				documents = append(documents, sourceDoc)
			}
		}
	}
	// =====================================================================================

	log.Printf("SERVICE-HELPER: Retrieved %d documents", len(documents))
	return documents, nil
}

// generateResponseWithGemini generates a response using a Gemini Chat Session
func (r *ragServiceImpl) generateResponseWithGemini(c context.Context, chatSession *genai.Chat, prompt string) (string, error) {
	log.Printf("SERVICE-HELPER: Sending prompt to Gemini with tool support using Chat Session...")

	// 1. Define the initial message to send. This is the first turn of the conversation.
	currentPart := genai.Part{Text: prompt}

	// 2. Loop to handle potential multi-turn interactions (like function calls).
	for {
		// 3. Send the current part to the model. The chatSession object automatically includes
		// the entire conversation history from previous turns.
		result, err := chatSession.SendMessage(c, currentPart)
		if err != nil {
			return "", fmt.Errorf("gemini api call failed: %w", err)
		}

		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return "I'm sorry, I couldn't generate a response based on the provided notes.", nil
		}

		// Extract the first part of the model's response.
		part := result.Candidates[0].Content.Parts[0]

		// 4. Check if the model requested a function call.
		if part.FunctionCall != nil {
			call := part.FunctionCall
			log.Printf("Gemini wants to call function: %s with args: %v", call.Name, call.Args)

			var resultStr string
			switch call.Name {
			case "createMarkdownFile":
				resultStr = r.FileActions.CreateMarkdownFile(call.Args["filename"].(string), call.Args["content"].(string))
			case "deleteMarkdownFile":
				resultStr = r.FileActions.DeleteMarkdownFile(call.Args["filename"].(string))
			case "editMarkdownFile":
				resultStr = r.FileActions.EditMarkdownFile(call.Args["filename"].(string), call.Args["content"].(string))
			default:
				resultStr = fmt.Sprintf("Error: Unknown function '%s' requested.", call.Name)
			}

			// 5. Prepare the function's result to be sent back to the model in the next turn.
			// We set `currentPart` to this FunctionResponse.
			currentPart = genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     call.Name,
					Response: map[string]interface{}{"result": resultStr},
				},
			}

			// 6. Continue the loop to send the function result back and get the next response.
			continue
		}

		// 7. If it's not a function call, we have our final text answer.
		var responseText strings.Builder
		for _, p := range result.Candidates[0].Content.Parts {
			if p.Text != "" {
				responseText.WriteString(p.Text)
			}
		}
		return responseText.String(), nil
	}
}

// createRAGPrompt creates a prompt with context for the LLM
func (r *ragServiceImpl) createRAGPrompt(query string, retrievedDocs []models.SourceDocument) string {
	// If no relevant documents are found, just pass the user's query directly.
	// The model will rely on its conversational memory.
	if len(retrievedDocs) == 0 {
		return query
	}

	// If documents are found, build a prompt that includes them as context.
	var context strings.Builder
	context.WriteString("Use the following context to answer the question. If the answer is not in the context, use our previous conversation history.\n\n")
	context.WriteString("Context:\n")
	for _, doc := range retrievedDocs {
		context.WriteString(fmt.Sprintf("- %s\n", doc.Text))
	}

	// This new prompt structure gives the model the flexibility it needs.
	prompt := fmt.Sprintf("%s\nQuestion: %s\n\nAnswer:", context.String(), query)
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
func NewRAGService(client *http.Client, collection chromago.Collection, geminiClient *genai.Client, fileActions *FileActions) RAGService {
	return &ragServiceImpl{
		httpClient:   client,
		collection:   collection, // No longer a pointer
		geminiClient: geminiClient,
		FileActions:  fileActions, // Initialize FileActions
		chatSessions: make(map[string]*genai.Chat),
	}
}
