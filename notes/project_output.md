# Project: 

## Project Structure

```
- /
    - README.md
    - .gitignore
    - endpoints.json
- server/
    - .env
    - main.go
    - middleware/
    - models/
        - notes.go
        - response.go
        - request.go
        - ollama.go
    - controller/
        - rag_controller.go
    - services/
        - rag_service.go
- notes/
    - scripts/
        - markdown.py
- client/
    - index.html
    - README.md
    - .gitignore
    - public/
        - vite.svg
    - src/
        - App.css
        - theme.js
        - index.css
        - main.jsx
        - App.jsx
        - components/
            - Header.jsx
            - QueryResults.jsx
            - Footer.jsx
            - QueryInput.jsx
            - NoteList.jsx
            - NoteInputForm.jsx
        - assets/
            - react.svg

```

## File: `README.md`

```md
# Local RAG API with Go, Gemini, and ChromaDB

This project implements a Retrieval-Augmented Generation (RAG) API using Go, Google's Gemini, ChromaDB for vector storage, and a local Ollama instance for generating embeddings.

## Prerequisites

Before you begin, ensure you have the following installed:
- [Go](https://go.dev/doc/install) (version 1.21 or later)
- [Python](https://www.python.org/downloads/) & [pip](https://pip.pypa.io/en/stable/installation/)
- [Ollama](https://ollama.ai/)

## 1. Setup

### a. Get Your Gemini API Key

1. Go to the [Google AI Studio](https://aistudio.google.com/).
2. Sign in with your Google account.
3. Click on **"Get API key"** and create a new API key in a new or existing project.
4. Copy the generated API key.

### b. Set Environment Variable

You need to set the `GEMINI_API_KEY` environment variable for the Go application to authenticate with the Gemini API. You can do this by exporting it in your shell configuration file (e.g., `.zshrc`, `.bashrc`) or by setting it in the terminal session where you'll run the server.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
```
Replace `"YOUR_GEMINI_API_KEY"` with the key you obtained.

## 2. Running the Services

The application requires three separate services to be running: ChromaDB, Ollama, and the Go backend server.

### a. Run ChromaDB Locally

First, you need to install the `chromadb` Python package.

```bash
pip install chromadb
```

Once installed, open a new terminal window and run the following command to start the ChromaDB server. This command tells Chroma to persist its data to the `./my_chroma_data` directory, which will be created if it doesn't exist.

```bash
chroma run --path ./my_chroma_data
```
Keep this process running.

### b. Run the Ollama Server

In another terminal window, start the Ollama server. This will host the local model needed for generating text embeddings.

```bash
ollama serve
```

Once the server is running, open a **third** terminal window and pull the `nomic-embed-text` model. This is the specific model the Go application is configured to use for embeddings.

```bash
ollama pull nomic-embed-text:v1.5
```
Keep the `ollama serve` process running.

### c. Run the Go Backend Server

Finally, navigate to the `server` directory of the project and start the Go application.

```bash
cd server
go run main.go
```

You should see log messages indicating that the server has started successfully on port 8080 and has connected to the Gemini and ChromaDB services.

## 3. Using the API

The API provides endpoints to ingest notes, ask questions, and retrieve all stored notes. You can use a tool like `curl` or Postman to interact with the API.

Refer to the `endpoints.json` file for a detailed list of endpoints and their corresponding JSON payloads.

### Example `curl` Commands

#### Ingest a Note
```bash
curl -X POST http://localhost:8080/api/v1/notes \
-H "Content-Type: application/json" \
-d '{ 
  "text": "The sky is blue during a clear day."
}'
```


#### Query the RAG Pipeline
```bash
curl -X POST http://localhost:8080/api/v1/query \
-H "Content-Type: application/json" \
-d '{ "query": "What color is the sky?" }'
```


#### Get All Ingested Notes
```bash
curl -X GET http://localhost:8080/api/v1/notes
```

#### Health Check
```bash
curl -X GET http://localhost:8080/health
```


```

## File: `.gitignore`

```
server/.env
my_chroma_data/
my_chroma_data/chroma.sqlite3
```

## File: `endpoints.json`

```json
[
  {
    "endpoint": "/api/v1/notes",
    "method": "POST",
    "payload": {
      "text": "This is a test note to be ingested into ChromaDB."
    }
  },
  {
    "endpoint": "/api/v1/query",
    "method": "POST",
    "payload": {
      "query": "What is the content of the test note?"
    }
  },
  {
    "endpoint": "/api/v1/notes",
    "method": "GET",
    "payload": null
  },
  {
    "endpoint": "/health",
    "method": "GET",
    "payload": null
  }
]

```

## File: `server/.env`

```
GEMINI_API_KEY=''
OLLAMA_URL=http://localhost:11434/api/embeddings
OLLAMA_MODEL=nomic-embed-text
GEMINI_MODEL=gemini-1.5-flash
RETRIEVAL_N_RESULTS=3

```

## File: `server/main.go`

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github/itish2003/rag/controller"
	"github/itish2003/rag/services"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

func main() {
	// Create HTTP client properly
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Chroma client using v2 API
	chromaClient, err := chromago.NewHTTPClient()
	if err != nil {
		log.Fatalf("FATAL: Failed to create chroma client: %v", err)
	}

	// Ensure we close the client to release resources like local embedding functions
	defer func() {
		if errorss := chromaClient.Close(); errorss != nil {
			log.Printf("Warning: Failed to close chroma client: %v", errorss)
		}
	}()

	// Get or create collection using v2 API
	collection, err := getOrCreateCollectionV2(chromaClient, "test-collection")
	if err != nil {
		log.Fatalf("FATAL: Failed to get or create collection: %v", err)
	}

	// Create Gemini client
	geminiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("FATAL: Failed to create Gemini client: %v. Make sure GEMINI_API_KEY is set.", err)
	}
	log.Println("Successfully connected to Google Gemini.")

	// Use the proper constructor function
	ragService := services.NewRAGService(httpClient, collection, geminiClient)
	ragController := controller.NewRAGController(ragService)

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware for testing
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Add health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "RAG API",
			"version": "1.0.0",
		})
	})

	// API routes
	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/notes", ragController.IngestNote) // Endpoint to create a new note
		apiV1.GET("/notes", ragController.GetAllNotes) // Endpoint to get all notes
		apiV1.POST("/query", ragController.QueryRAG)   // Endpoint to ask a question
	}

	// Start the Server
	port := "8080"
	log.Printf("Go Gin backend server starting on http://localhost:%s", port)
	log.Printf("Health check available at: http://localhost:%s/health", port)
	log.Printf("API endpoints:")
	log.Printf("  POST http://localhost:%s/api/v1/notes", port)
	log.Printf("  POST http://localhost:%s/api/v1/query", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}

// getOrCreateCollectionV2 implements collection management using v2 API
func getOrCreateCollectionV2(client chromago.Client, collectionName string) (chromago.Collection, error) {
	ctx := context.Background()

	log.Printf("Getting or creating collection '%s' using v2 API...", collectionName)

	// Use v2 API's GetOrCreateCollection method
	collection, err := client.GetOrCreateCollection(
		ctx,
		collectionName,
		chromago.WithCollectionMetadataCreate(
			chromago.NewMetadata(
				chromago.NewStringAttribute("description", "RAG application collection"),
				chromago.NewStringAttribute("created_by", "rag_service"),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Successfully got/created collection '%s'", collectionName)
	return collection, nil
}

```

## File: `server/models/notes.go`

```go
package models

// Note represents a single document retrieved from the vector database.
type Note struct {
	ID       string                 `json:"id"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetAllNotesResponse is the structure for the response of the GET /notes endpoint.
type GetAllNotesResponse struct {
	Count int    `json:"count"`
	Notes []Note `json:"notes"`
}

```

## File: `server/models/response.go`

```go
package models

type InjestDataResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type QueryRAGResponse struct {
	Answer     string   `json:"answer"`
	SourceDocs []string `json:"source_docs,omitempty"`
	Error      string   `json:"error,omitempty"`
}

```

## File: `server/models/request.go`

```go
package models

type IngestDataRequest struct {
	Text string `json:"text"`
}

type QueryTextRequest struct {
	Query string `json:"query"`
}

```

## File: `server/models/ollama.go`

```go
package models

// OllamaEmbedRequest is used to structure the request to the Ollama embedding API.
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse is used to parse the embedding from the Ollama API response.
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

```

## File: `server/controller/rag_controller.go`

```go
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	// Import your local packages using your module path
	"github/itish2003/rag/models"
	"github/itish2003/rag/services"
)

// RAGController handles the HTTP requests for our RAG API. It depends on the
// RAGService to perform the actual business logic.
type RAGController struct {
	ragService services.RAGService
}

// NewRAGController is a constructor function that creates a new RAGController.
// This is called from main.go to inject the service dependency.
func NewRAGController(service services.RAGService) *RAGController {
	return &RAGController{
		ragService: service,
	}
}

// IngestNote is the Gin handler for the POST /api/v1/notes endpoint.
// It parses the request, calls the service layer, and returns the HTTP response.
func (c *RAGController) IngestNote(ctx *gin.Context) {
	var req models.IngestDataRequest

	// Use Gin's binding to parse and validate the incoming JSON.
	// This will bind the request body to our `req` struct.
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Delegate the core logic to the service layer.
	// We extract the standard context from Gin's context for portability.
	if err := c.ragService.IngestNote(ctx.Request.Context(), req); err != nil {
		// If the service returns an error, respond with a generic server error.
		// The actual error should be logged by the service layer.
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ingest note"})
		return
	}

	// On success, return a 201 Created status and a success message.
	ctx.JSON(http.StatusCreated, gin.H{"message": "Note ingested successfully"})
}

// QueryRAG is the Gin handler for the POST /api/v1/query endpoint.
// It orchestrates the RAG pipeline by calling the service layer.
func (c *RAGController) QueryRAG(ctx *gin.Context) {
	var req models.QueryTextRequest

	// Bind the request JSON to our QueryTextRequest struct.
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Delegate the complex RAG pipeline logic to the service layer.
	// The service will return the final response object or an error.
	response, err := c.ragService.QueryRAG(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate AI response"})
		return
	}

	// On success, return a 200 OK status with the response data from the service.
	ctx.JSON(http.StatusOK, response)
}

// GetAllNotes is the Gin handler for the GET /api/v1/notes endpoint.
func (c *RAGController) GetAllNotes(ctx *gin.Context) {
	// Delegate the logic to the service layer.
	response, err := c.ragService.GetAllNotes(ctx.Request.Context())
	if err != nil {
		// If the service returns an error, respond with a generic server error.
		// The actual error should be logged by the service layer.
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve notes"})
		return
	}

	// On success, return a 200 OK status with the response data from the service.
	ctx.JSON(http.StatusOK, response)
}

```

## File: `server/services/rag_service.go`

```go
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

	embeddingVector, err := r.embedTextWithOllama(c, req.Text)
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
	queryEmbedding, err := r.embedTextWithOllama(c, query)
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

// embedTextWithOllama generates embeddings using Ollama
func (r *ragServiceImpl) embedTextWithOllama(c context.Context, textToEmbed string) ([]float32, error) {
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

```

## File: `notes/scripts/markdown.py`

```py
import os

def get_project_structure(project_path, ignore_list=None):
    """
    Recursively walks through a directory and returns a string
    representing the directory structure.
    """
    if ignore_list is None:
        ignore_list = []
    structure = ""
    for root, dirs, files in os.walk(project_path):
        # Filter out ignored directories
        dirs[:] = [d for d in dirs if d not in ignore_list]
        
        level = root.replace(project_path, '').count(os.sep)
        indent = ' ' * 4 * level
        structure += f"{indent}- {os.path.basename(root)}/\n"
        sub_indent = ' ' * 4 * (level + 1)
        for f in files:
            if f not in ignore_list:
                structure += f"{sub_indent}- {f}\n"
    return structure

def create_markdown_from_project(project_path, output_file, ignore_list=None):
    """
    Reads all files in a project and writes their content into a single
    markdown file.
    """
    if ignore_list is None:
        ignore_list = []

    with open(output_file, 'w', encoding='utf-8') as md_file:
        md_file.write(f"# Project: {os.path.basename(project_path)}\n\n")

        # Add the project structure to the markdown file
        md_file.write("## Project Structure\n\n")
        project_structure = get_project_structure(project_path, ignore_list)
        md_file.write(f"```\n{project_structure}\n```\n\n")

        for root, dirs, files in os.walk(project_path):
            # Filter out ignored directories
            dirs[:] = [d for d in dirs if d not in ignore_list]

            for file_name in files:
                if file_name in ignore_list:
                    continue
                file_path = os.path.join(root, file_name)
                relative_path = os.path.relpath(file_path, project_path)
                md_file.write(f"## File: `{relative_path}`\n\n")
                
                try:
                    with open(file_path, 'r', encoding='utf-8', errors='ignore') as file_content:
                        content = file_content.read()
                        file_extension = os.path.splitext(file_name)[1].lstrip('.')
                        md_file.write(f"```{file_extension}\n")
                        md_file.write(content)
                        md_file.write("\n```\n\n")
                except Exception as e:
                    md_file.write(f"Could not read file: {e}\n\n")

if __name__ == '__main__':
    # Replace with the path to your project
    project_directory = './' 
    # The name of the output markdown file
    output_markdown_file = 'project_output.md' 
    
    # Optional: specify files and directories to ignore
    files_and_dirs_to_ignore = ['.git', '__pycache__', '.vscode', 'node_modules', 'my_chroma_data', 'go.mod','package.json','package-lock.json','vite.config.js','eslint.config.js', 'go.sum', output_markdown_file]

    create_markdown_from_project(project_directory, output_markdown_file, files_and_dirs_to_ignore)
    print(f"Project content has been written to {output_markdown_file}")
```

## File: `client/index.html`

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + React</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>

```

## File: `client/README.md`

```md
# React + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## Expanding the ESLint configuration

If you are developing a production application, we recommend using TypeScript with type-aware lint rules enabled. Check out the [TS template](https://github.com/vitejs/vite/tree/main/packages/create-vite/template-react-ts) for information on how to integrate TypeScript and [`typescript-eslint`](https://typescript-eslint.io) in your project.

```

## File: `client/.gitignore`

```
# Logs
logs
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*
lerna-debug.log*

node_modules
dist
dist-ssr
*.local

# Editor directories and files
.vscode/*
!.vscode/extensions.json
.idea
.DS_Store
*.suo
*.ntvs*
*.njsproj
*.sln
*.sw?

```

## File: `client/public/vite.svg`

```svg
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" role="img" class="iconify iconify--logos" width="31.88" height="32" preserveAspectRatio="xMidYMid meet" viewBox="0 0 256 257"><defs><linearGradient id="IconifyId1813088fe1fbc01fb466" x1="-.828%" x2="57.636%" y1="7.652%" y2="78.411%"><stop offset="0%" stop-color="#41D1FF"></stop><stop offset="100%" stop-color="#BD34FE"></stop></linearGradient><linearGradient id="IconifyId1813088fe1fbc01fb467" x1="43.376%" x2="50.316%" y1="2.242%" y2="89.03%"><stop offset="0%" stop-color="#FFEA83"></stop><stop offset="8.333%" stop-color="#FFDD35"></stop><stop offset="100%" stop-color="#FFA800"></stop></linearGradient></defs><path fill="url(#IconifyId1813088fe1fbc01fb466)" d="M255.153 37.938L134.897 252.976c-2.483 4.44-8.862 4.466-11.382.048L.875 37.958c-2.746-4.814 1.371-10.646 6.827-9.67l120.385 21.517a6.537 6.537 0 0 0 2.322-.004l117.867-21.483c5.438-.991 9.574 4.796 6.877 9.62Z"></path><path fill="url(#IconifyId1813088fe1fbc01fb467)" d="M185.432.063L96.44 17.501a3.268 3.268 0 0 0-2.634 3.014l-5.474 92.456a3.268 3.268 0 0 0 3.997 3.378l24.777-5.718c2.318-.535 4.413 1.507 3.936 3.838l-7.361 36.047c-.495 2.426 1.782 4.5 4.151 3.78l15.304-4.649c2.372-.72 4.652 1.36 4.15 3.788l-11.698 56.621c-.732 3.542 3.979 5.473 5.943 2.437l1.313-2.028l72.516-144.72c1.215-2.423-.88-5.186-3.54-4.672l-25.505 4.922c-2.396.462-4.435-1.77-3.759-4.114l16.646-57.705c.677-2.35-1.37-4.583-3.769-4.113Z"></path></svg>
```

## File: `client/src/App.css`

```css
#root {
  max-width: 1280px;
  margin: 0 auto;
  padding: 2rem;
  text-align: center;
}

.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
  transition: filter 300ms;
}
.logo:hover {
  filter: drop-shadow(0 0 2em #646cffaa);
}
.logo.react:hover {
  filter: drop-shadow(0 0 2em #61dafbaa);
}

@keyframes logo-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: no-preference) {
  a:nth-of-type(2) .logo {
    animation: logo-spin infinite 20s linear;
  }
}

.card {
  padding: 2em;
}

.read-the-docs {
  color: #888;
}

```

## File: `client/src/theme.js`

```js
import { createTheme } from '@mui/material/styles';

// A clean, professional, and minimalist light theme.
const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1976d2', // A classic, professional blue
    },
    secondary: {
      main: '#dc004e', // A contrasting pink for secondary actions if needed
    },
    background: {
      default: '#f4f6f8', // A very light, soft gray
      paper: '#ffffff',   // Pure white for cards and surfaces
    },
    text: {
      primary: '#212121',   // Crisp, dark gray for high readability
      secondary: '#757575', // Lighter gray for secondary text
    },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    h6: {
      fontWeight: 600,
    },
  },
  components: {
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: '#ffffff',
          color: '#212121', // Dark text on a light app bar
          boxShadow: '0 1px 4px rgba(0,0,0,0.1)', // A subtle shadow for depth
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: '8px',
          border: '1px solid #e0e0e0', // A light border for definition
          boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
          borderRadius: '6px',
        },
      },
    },
  },
});

export default theme;

```

## File: `client/src/index.css`

```css
:root {
  font-family: system-ui, Avenir, Helvetica, Arial, sans-serif;
  line-height: 1.5;
  font-weight: 400;

  color-scheme: light dark;
  color: rgba(255, 255, 255, 0.87);
  background-color: #242424;

  font-synthesis: none;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

a {
  font-weight: 500;
  color: #646cff;
  text-decoration: inherit;
}
a:hover {
  color: #535bf2;
}

body {
  margin: 0;
  display: flex;
  place-items: center;
  min-width: 320px;
  min-height: 100vh;
}

h1 {
  font-size: 3.2em;
  line-height: 1.1;
}

button {
  border-radius: 8px;
  border: 1px solid transparent;
  padding: 0.6em 1.2em;
  font-size: 1em;
  font-weight: 500;
  font-family: inherit;
  background-color: #1a1a1a;
  cursor: pointer;
  transition: border-color 0.25s;
}
button:hover {
  border-color: #646cff;
}
button:focus,
button:focus-visible {
  outline: 4px auto -webkit-focus-ring-color;
}

@media (prefers-color-scheme: light) {
  :root {
    color: #213547;
    background-color: #ffffff;
  }
  a:hover {
    color: #747bff;
  }
  button {
    background-color: #f9f9f9;
  }
}

```

## File: `client/src/main.jsx`

```jsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import theme from './theme';

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <App />
    </ThemeProvider>
  </StrictMode>,
)

```

## File: `client/src/App.jsx`

```jsx
import Header from './components/Header';
import Footer from './components/Footer';
import NoteInputForm from './components/NoteInputForm';
import NoteList from './components/NoteList';
import QueryInput from './components/QueryInput';
import QueryResults from './components/QueryResults';
import Box from '@mui/material/Box';
import Container from '@mui/material/Container';
import Paper from '@mui/material/Paper';
import { useState } from 'react';

function App() {
  const [refreshNotes, setRefreshNotes] = useState(0);
  const [queryResult, setQueryResult] = useState(null);
  const [queryLoading, setQueryLoading] = useState(false);
  const [queryError, setQueryError] = useState('');

  const handleNoteAdded = () => setRefreshNotes(r => r + 1);

  const handleQuery = async (query) => {
    setQueryLoading(true);
    setQueryError('');
    setQueryResult(null);
    try {
      const res = await fetch('/api/v1/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query }),
      });
      if (!res.ok) throw new Error('Query failed');
      const data = await res.json();
      setQueryResult(data);
    } catch (err) {
      setQueryError('Failed to get results.');
    } finally {
      setQueryLoading(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header />
      <Container maxWidth="lg" sx={{ flex: 1, py: 4 }}>
        <Paper sx={{ display: 'flex', gap: 4, p: 2, minHeight: 500 }}>
          {/* Left Column: Note Ingestion */}
          <Box sx={{ flex: 1, pr: 2, borderRight: (theme) => `1px solid ${theme.palette.divider}` }}>
            <NoteInputForm onNoteAdded={handleNoteAdded} />
            <NoteList refresh={refreshNotes} />
          </Box>
          {/* Right Column: Query and Results */}
          <Box sx={{ flex: 2, pl: 2 }}>
            <QueryInput onQuery={handleQuery} />
            <QueryResults result={queryResult} loading={queryLoading} error={queryError} />
          </Box>
        </Paper>
      </Container>
      <Footer />
    </Box>
  );
}

export default App;

```

## File: `client/src/components/Header.jsx`

```jsx
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';

function Header() {
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static" color="primary">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Local RAG UI
          </Typography>
        </Toolbar>
      </AppBar>
    </Box>
  );
}

export default Header;

```

## File: `client/src/components/QueryResults.jsx`

```jsx
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';

function QueryResults({ result, loading, error }) {
  if (loading) return <Typography>Loading...</Typography>;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!result) return null;

  return (
    <Card sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 1 }}>Answer</Typography>
        <Typography sx={{ mb: 2 }}>{result.answer || result.response || 'No answer.'}</Typography>
        {result.sources && result.sources.length > 0 && (
          <Box>
            <Typography variant="subtitle2">Source Documents:</Typography>
            <ul>
              {result.sources.map((src, idx) => (
                <li key={idx}>
                  <Typography variant="body2">{src}</Typography>
                </li>
              ))}
            </ul>
          </Box>
        )}
      </CardContent>
    </Card>
  );
}

export default QueryResults;

```

## File: `client/src/components/Footer.jsx`

```jsx
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

function Footer() {
  return (
    <Box component="footer" sx={{ mt: 4, py: 2, textAlign: 'center', bgcolor: 'background.paper' }}>
      <Typography variant="body2" color="text.secondary">
        <a href="https://github.com/your-repo" target="_blank" rel="noopener noreferrer">
          View on GitHub
        </a>
      </Typography>
    </Box>
  );
}

export default Footer;

```

## File: `client/src/components/QueryInput.jsx`

```jsx
import { useState } from 'react';
import Box from '@mui/material/Box';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import InputAdornment from '@mui/material/InputAdornment';
import SearchIcon from '@mui/icons-material/Search';
import CircularProgress from '@mui/material/CircularProgress';

function QueryInput({ onQuery }) {
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!query.trim()) {
      setError('Query cannot be empty.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      await onQuery(query);
    } catch {
      setError('Failed to get results.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ mb: 2 }}>
      <TextField
        label="Ask a question"
        fullWidth
        value={query}
        onChange={e => setQuery(e.target.value)}
        error={!!error}
        helperText={error}
        disabled={loading}
        InputProps={{
          endAdornment: (
            <InputAdornment position="end">
              <SearchIcon />
            </InputAdornment>
          ),
        }}
        sx={{ mb: 2 }}
      />
      <Button type="submit" variant="contained" disabled={loading}>
        {loading ? <CircularProgress size={24} /> : 'Search'}
      </Button>
    </Box>
  );
}

export default QueryInput;

```

## File: `client/src/components/NoteList.jsx`

```jsx
import { useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import CircularProgress from '@mui/material/CircularProgress';

function NoteList({ refresh }) {
  const [notes, setNotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/notes')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch notes');
        return res.json();
      })
      .then(data => {
        setNotes(data.notes || []);
        setError('');
      })
      .catch(() => setError('Failed to load notes.'))
      .finally(() => setLoading(false));
  }, [refresh]);

  if (loading) return <CircularProgress sx={{ display: 'block', mx: 'auto', my: 2 }} />;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!notes.length) return <Typography>No notes yet. Add your first note!</Typography>;

  return (
    <Box sx={{ maxHeight: 300, overflowY: 'auto' }}>
      {notes.map((note) => (
        <Card key={note.id} sx={{ mb: 1 }}>
          <CardContent>
            <Typography>{note.text}</Typography>
          </CardContent>
        </Card>
      ))}
    </Box>
  );
}

export default NoteList;

```

## File: `client/src/components/NoteInputForm.jsx`

```jsx
import { useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

function NoteInputForm({ onNoteAdded }) {
  const [note, setNote] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!note.trim()) {
      setError('Note cannot be empty.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const res = await fetch('/api/v1/notes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: note }),
      });
      if (!res.ok) throw new Error('Failed to add note');
      setNote('');
      onNoteAdded();
    } catch (err) {
      setError('Failed to add note.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField
            label="Add a new note"
            multiline
            minRows={2}
            fullWidth
            value={note}
            onChange={e => setNote(e.target.value)}
            error={!!error}
            helperText={error}
            disabled={loading}
            sx={{ mb: 2 }}
          />
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? <CircularProgress size={24} /> : 'Add Note'}
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
}

export default NoteInputForm;

```

## File: `client/src/assets/react.svg`

```svg
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" role="img" class="iconify iconify--logos" width="35.93" height="32" preserveAspectRatio="xMidYMid meet" viewBox="0 0 256 228"><path fill="#00D8FF" d="M210.483 73.824a171.49 171.49 0 0 0-8.24-2.597c.465-1.9.893-3.777 1.273-5.621c6.238-30.281 2.16-54.676-11.769-62.708c-13.355-7.7-35.196.329-57.254 19.526a171.23 171.23 0 0 0-6.375 5.848a155.866 155.866 0 0 0-4.241-3.917C100.759 3.829 77.587-4.822 63.673 3.233C50.33 10.957 46.379 33.89 51.995 62.588a170.974 170.974 0 0 0 1.892 8.48c-3.28.932-6.445 1.924-9.474 2.98C17.309 83.498 0 98.307 0 113.668c0 15.865 18.582 31.778 46.812 41.427a145.52 145.52 0 0 0 6.921 2.165a167.467 167.467 0 0 0-2.01 9.138c-5.354 28.2-1.173 50.591 12.134 58.266c13.744 7.926 36.812-.22 59.273-19.855a145.567 145.567 0 0 0 5.342-4.923a168.064 168.064 0 0 0 6.92 6.314c21.758 18.722 43.246 26.282 56.54 18.586c13.731-7.949 18.194-32.003 12.4-61.268a145.016 145.016 0 0 0-1.535-6.842c1.62-.48 3.21-.974 4.76-1.488c29.348-9.723 48.443-25.443 48.443-41.52c0-15.417-17.868-30.326-45.517-39.844Zm-6.365 70.984c-1.4.463-2.836.91-4.3 1.345c-3.24-10.257-7.612-21.163-12.963-32.432c5.106-11 9.31-21.767 12.459-31.957c2.619.758 5.16 1.557 7.61 2.4c23.69 8.156 38.14 20.213 38.14 29.504c0 9.896-15.606 22.743-40.946 31.14Zm-10.514 20.834c2.562 12.94 2.927 24.64 1.23 33.787c-1.524 8.219-4.59 13.698-8.382 15.893c-8.067 4.67-25.32-1.4-43.927-17.412a156.726 156.726 0 0 1-6.437-5.87c7.214-7.889 14.423-17.06 21.459-27.246c12.376-1.098 24.068-2.894 34.671-5.345a134.17 134.17 0 0 1 1.386 6.193ZM87.276 214.515c-7.882 2.783-14.16 2.863-17.955.675c-8.075-4.657-11.432-22.636-6.853-46.752a156.923 156.923 0 0 1 1.869-8.499c10.486 2.32 22.093 3.988 34.498 4.994c7.084 9.967 14.501 19.128 21.976 27.15a134.668 134.668 0 0 1-4.877 4.492c-9.933 8.682-19.886 14.842-28.658 17.94ZM50.35 144.747c-12.483-4.267-22.792-9.812-29.858-15.863c-6.35-5.437-9.555-10.836-9.555-15.216c0-9.322 13.897-21.212 37.076-29.293c2.813-.98 5.757-1.905 8.812-2.773c3.204 10.42 7.406 21.315 12.477 32.332c-5.137 11.18-9.399 22.249-12.634 32.792a134.718 134.718 0 0 1-6.318-1.979Zm12.378-84.26c-4.811-24.587-1.616-43.134 6.425-47.789c8.564-4.958 27.502 2.111 47.463 19.835a144.318 144.318 0 0 1 3.841 3.545c-7.438 7.987-14.787 17.08-21.808 26.988c-12.04 1.116-23.565 2.908-34.161 5.309a160.342 160.342 0 0 1-1.76-7.887Zm110.427 27.268a347.8 347.8 0 0 0-7.785-12.803c8.168 1.033 15.994 2.404 23.343 4.08c-2.206 7.072-4.956 14.465-8.193 22.045a381.151 381.151 0 0 0-7.365-13.322Zm-45.032-43.861c5.044 5.465 10.096 11.566 15.065 18.186a322.04 322.04 0 0 0-30.257-.006c4.974-6.559 10.069-12.652 15.192-18.18ZM82.802 87.83a323.167 323.167 0 0 0-7.227 13.238c-3.184-7.553-5.909-14.98-8.134-22.152c7.304-1.634 15.093-2.97 23.209-3.984a321.524 321.524 0 0 0-7.848 12.897Zm8.081 65.352c-8.385-.936-16.291-2.203-23.593-3.793c2.26-7.3 5.045-14.885 8.298-22.6a321.187 321.187 0 0 0 7.257 13.246c2.594 4.48 5.28 8.868 8.038 13.147Zm37.542 31.03c-5.184-5.592-10.354-11.779-15.403-18.433c4.902.192 9.899.29 14.978.29c5.218 0 10.376-.117 15.453-.343c-4.985 6.774-10.018 12.97-15.028 18.486Zm52.198-57.817c3.422 7.8 6.306 15.345 8.596 22.52c-7.422 1.694-15.436 3.058-23.88 4.071a382.417 382.417 0 0 0 7.859-13.026a347.403 347.403 0 0 0 7.425-13.565Zm-16.898 8.101a358.557 358.557 0 0 1-12.281 19.815a329.4 329.4 0 0 1-23.444.823c-7.967 0-15.716-.248-23.178-.732a310.202 310.202 0 0 1-12.513-19.846h.001a307.41 307.41 0 0 1-10.923-20.627a310.278 310.278 0 0 1 10.89-20.637l-.001.001a307.318 307.318 0 0 1 12.413-19.761c7.613-.576 15.42-.876 23.31-.876H128c7.926 0 15.743.303 23.354.883a329.357 329.357 0 0 1 12.335 19.695a358.489 358.489 0 0 1 11.036 20.54a329.472 329.472 0 0 1-11 20.722Zm22.56-122.124c8.572 4.944 11.906 24.881 6.52 51.026c-.344 1.668-.73 3.367-1.15 5.09c-10.622-2.452-22.155-4.275-34.23-5.408c-7.034-10.017-14.323-19.124-21.64-27.008a160.789 160.789 0 0 1 5.888-5.4c18.9-16.447 36.564-22.941 44.612-18.3ZM128 90.808c12.625 0 22.86 10.235 22.86 22.86s-10.235 22.86-22.86 22.86s-22.86-10.235-22.86-22.86s10.235-22.86 22.86-22.86Z"></path></svg>
```

package v2

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Collection interface {
	// Name returns the name of the collection
	Name() string
	// ID returns the id of the collection
	ID() string
	// Tenant returns the tenant of the collection
	Tenant() Tenant
	// Database returns the database of the collection
	Database() Database
	// Metadata returns the metadata of the collection
	Metadata() CollectionMetadata
	// Dimension returns the dimension of the embeddings in the collection
	Dimension() int
	// Configuration returns the configuration of the collection
	Configuration() CollectionConfiguration
	// Add adds a document to the collection
	Add(ctx context.Context, opts ...CollectionAddOption) error
	// Upsert updates or adds a document to the collection
	Upsert(ctx context.Context, opts ...CollectionAddOption) error
	// Update updates a document in the collection
	Update(ctx context.Context, opts ...CollectionUpdateOption) error
	// Delete deletes documents from the collection
	Delete(ctx context.Context, opts ...CollectionDeleteOption) error
	// Count returns the number of documents in the collection
	Count(ctx context.Context) (int, error)
	// ModifyName modifies the name of the collection
	ModifyName(ctx context.Context, newName string) error
	// ModifyMetadata modifies the metadata of the collection
	ModifyMetadata(ctx context.Context, newMetadata CollectionMetadata) error
	// ModifyConfiguration modifies the configuration of the collection
	ModifyConfiguration(ctx context.Context, newConfig CollectionConfiguration) error // not supported yet
	// Get gets documents from the collection
	Get(ctx context.Context, opts ...CollectionGetOption) (GetResult, error)
	// Query queries the collection
	Query(ctx context.Context, opts ...CollectionQueryOption) (QueryResult, error)
	// Close closes the collection and releases any resources
	Close() error
}

type CollectionOp interface {
	// PrepareAndValidate validates the operation. Each operation must implement this method to ensure the operation is valid and can be sent over the wire
	PrepareAndValidate() error
	EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error
	// MarshalJSON marshals the operation to JSON
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals the operation from JSON
	UnmarshalJSON(b []byte) error
}

type FilterOp struct {
	Where         WhereFilter         `json:"where,omitempty"`
	WhereDocument WhereDocumentFilter `json:"where_document,omitempty"`
}

type FilterIDOp struct {
	Ids []DocumentID `json:"ids,omitempty"`
}

type FilterTextsOp struct {
	QueryTexts []string `json:"-"`
}

type FilterEmbeddingsOp struct {
	QueryEmbeddings []embeddings.Embedding `json:"query_embeddings"`
}

type ProjectOp struct {
	Include []Include `json:"include,omitempty"`
}

type LimitAndOffsetOp struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type LimitResultOp struct {
	NResults int `json:"n_results"`
}

type SortOp struct {
	Sort string `json:"sort,omitempty"`
}

type CollectionGetOption func(get *CollectionGetOp) error

type CollectionGetOp struct {
	FilterOp          // ability to filter by where and whereDocument
	FilterIDOp        // ability to filter by id
	ProjectOp         // include metadatas, documents, embeddings, uris, ids
	LimitAndOffsetOp  // limit and offset
	SortOp            // sort
	ResourceOperation `json:"-"`
}

func NewCollectionGetOp(opts ...CollectionGetOption) (*CollectionGetOp, error) {
	get := &CollectionGetOp{
		ProjectOp: ProjectOp{Include: []Include{IncludeDocuments, IncludeMetadatas}},
	}
	for _, opt := range opts {
		err := opt(get)
		if err != nil {
			return nil, err
		}
	}
	return get, nil
}

func (c *CollectionGetOp) PrepareAndValidate() error {
	if c.Sort != "" {
		return errors.New("sort is not supported yet")
	}
	if c.Limit < 0 {
		return errors.New("limit must be greater than or equal to 0")
	}
	if c.Offset < 0 {
		return errors.New("offset must be greater than or equal to 0")
	}
	if len(c.Include) == 0 {
		return errors.New("at least one include option is required")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return err
		}
	}
	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *CollectionGetOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionGetOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionGetOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionGetOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionGetOp) Operation() OperationType {
	return OperationGet
}

func WithIDsGet(ids ...DocumentID) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		for _, id := range ids {
			query.Ids = append(query.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithWhereGet(where WhereFilter) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.Where = where
		return nil
	}
}

func WithWhereDocumentGet(whereDocument WhereDocumentFilter) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.WhereDocument = whereDocument
		return nil
	}
}

func WithIncludeGet(include ...Include) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.Include = include
		return nil
	}
}

func WithLimitGet(limit int) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		if limit <= 0 {
			return errors.New("limit must be greater than 0")
		}
		query.Limit = limit
		return nil
	}
}

func WithOffsetGet(offset int) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		if offset < 0 {
			return errors.New("offset must be greater than or equal to 0")
		}
		query.Offset = offset
		return nil
	}
}

// Query

type CollectionQueryOp struct {
	FilterOp
	FilterEmbeddingsOp
	FilterTextsOp
	LimitResultOp
	ProjectOp // include metadatas, documents, embeddings, uris
	FilterIDOp
}

func NewCollectionQueryOp(opts ...CollectionQueryOption) (*CollectionQueryOp, error) {
	query := &CollectionQueryOp{
		LimitResultOp: LimitResultOp{NResults: 10},
	}
	for _, opt := range opts {
		err := opt(query)
		if err != nil {
			return nil, err
		}
	}
	return query, nil
}

func (c *CollectionQueryOp) PrepareAndValidate() error {
	if len(c.QueryEmbeddings) == 0 && len(c.QueryTexts) == 0 {
		return errors.New("at least one query embedding or query text is required")
	}
	if c.NResults <= 0 {
		return errors.New("nResults must be greater than 0")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return errors.Wrap(err, "where validation failed")
		}
	}
	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CollectionQueryOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	if len(c.QueryTexts) > 0 && len(c.QueryEmbeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		embeddings, err := ef.EmbedDocuments(ctx, c.QueryTexts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		c.QueryEmbeddings = embeddings
	}
	return nil
}

func (c *CollectionQueryOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionQueryOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionQueryOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionQueryOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionQueryOp) Operation() OperationType {
	return OperationQuery
}

type CollectionQueryOption func(query *CollectionQueryOp) error

func WithWhereQuery(where WhereFilter) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.Where = where
		return nil
	}
}

func WithWhereDocumentQuery(whereDocument WhereDocumentFilter) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.WhereDocument = whereDocument
		return nil
	}
}

func WithNResults(nResults int) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if nResults <= 0 {
			return errors.New("nResults must be greater than 0")
		}
		query.NResults = nResults
		return nil
	}
}

func WithQueryTexts(queryTexts ...string) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(queryTexts) == 0 {
			return errors.New("at least one query text is required")
		}
		query.QueryTexts = queryTexts
		return nil
	}
}

func WithQueryEmbeddings(queryEmbeddings ...embeddings.Embedding) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(queryEmbeddings) == 0 {
			return errors.New("at least one query embedding is required")
		}
		query.QueryEmbeddings = queryEmbeddings
		return nil
	}
}

// WithIncludeQuery is used to include metadatas, documents, embeddings, uris in the query response.
func WithIncludeQuery(include ...Include) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.Include = include
		return nil
	}
}

// WithIDsQuery is used to filter the query by IDs. This is only available for Chroma version 1.0.3 and above.
func WithIDsQuery(ids ...DocumentID) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(ids) == 0 {
			return errors.New("at least one id is required")
		}
		if query.Ids == nil {
			query.Ids = make([]DocumentID, 0)
		}
		query.Ids = append(query.Ids, ids...)
		return nil
	}
}

// Add, Upsert, Update

type CollectionAddOp struct {
	Ids         []DocumentID           `json:"ids"`
	Documents   []Document             `json:"documents,omitempty"`
	Metadatas   []DocumentMetadata     `json:"metadatas,omitempty"`
	Embeddings  []embeddings.Embedding `json:"embeddings"`
	Records     []Record               `json:"-"`
	IDGenerator IDGenerator            `json:"-"`
}

func NewCollectionAddOp(opts ...CollectionAddOption) (*CollectionAddOp, error) {
	update := &CollectionAddOp{}
	for _, opt := range opts {
		err := opt(update)
		if err != nil {
			return nil, err
		}
	}
	return update, nil
}

func (c *CollectionAddOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	// invariants:
	// documents only - we embed
	// documents + embeddings - we skip
	// embeddings only - we skip
	if len(c.Documents) > 0 && len(c.Embeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		texts := make([]string, len(c.Documents))
		for i, doc := range c.Documents {
			texts[i] = doc.ContentString()
		}
		embeddings, err := ef.EmbedDocuments(ctx, texts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		for i, embedding := range embeddings {
			if i >= len(c.Embeddings) {
				c.Embeddings = append(c.Embeddings, embedding)
			} else {
				c.Embeddings[i] = embedding
			}
		}
	}
	return nil
}

func (c *CollectionAddOp) GenerateIDs() error {
	if c.IDGenerator == nil {
		return nil
	}
	generatedIDLen := 0
	switch {
	case len(c.Documents) > 0:
		generatedIDLen = len(c.Documents)
	case len(c.Embeddings) > 0:
		generatedIDLen = len(c.Embeddings)
	case len(c.Records) > 0:
		return errors.New("not implemented yet")
	default:
		return errors.New("at least one document or embedding is required")
	}
	c.Ids = make([]DocumentID, 0)
	for i := 0; i < generatedIDLen; i++ {
		switch {
		case len(c.Documents) > 0:
			c.Ids = append(c.Ids, DocumentID(c.IDGenerator.Generate(WithDocument(c.Documents[i].ContentString()))))
		case len(c.Embeddings) > 0:
			c.Ids = append(c.Ids, DocumentID(c.IDGenerator.Generate()))

		case len(c.Records) > 0:
			return errors.New("not implemented yet")
		}
	}
	return nil
}

func (c *CollectionAddOp) PrepareAndValidate() error {
	// invariants
	// - at least one ID or one record is required
	// - if IDs are provided, they must be unique
	// - if IDs are provided, the number of documents or embeddings must match the number of IDs
	// - if IDs are provided, if metadatas are also provided they must match the number of IDs

	if (len(c.Ids) == 0 && c.IDGenerator == nil) && len(c.Records) == 0 {
		return errors.New("at least one ID or record is required. Alternatively, an ID generator can be provided") // TODO add link to docs
	}

	// should we generate IDs?
	if c.IDGenerator != nil {
		err := c.GenerateIDs()
		if err != nil {
			return errors.Wrap(err, "failed to generate IDs")
		}
	}

	// if IDs are provided, they must be unique
	idSet := make(map[DocumentID]struct{})
	for _, id := range c.Ids {
		if _, exists := idSet[id]; exists {
			return errors.Errorf("duplicate id found: %s", id)
		}
		idSet[id] = struct{}{}
	}

	// if IDs are provided, the number of documents or embeddings must match the number of IDs
	if len(c.Documents) > 0 && len(c.Ids) != len(c.Documents) {
		return errors.Errorf("documents (%d) must match the number of ids (%d)", len(c.Documents), len(c.Ids))
	}

	if len(c.Embeddings) > 0 && len(c.Ids) != len(c.Embeddings) {
		return errors.Errorf("embeddings (%d) must match the number of ids (%d)", len(c.Embeddings), len(c.Ids))
	}

	// if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Metadatas) > 0 && len(c.Ids) != len(c.Metadatas) {
		return errors.Errorf("metadatas (%d) must match the number of ids (%d)", len(c.Metadatas), len(c.Ids))
	}

	if len(c.Records) > 0 {
		for _, record := range c.Records {
			err := record.Validate()
			if err != nil {
				return errors.Wrap(err, "record validation failed")
			}
			recordIds, recordDocuments, recordEmbeddings, recordMetadata := record.Unwrap()
			c.Ids = append(c.Ids, recordIds)
			c.Documents = append(c.Documents, recordDocuments)
			c.Metadatas = append(c.Metadatas, recordMetadata)
			c.Embeddings = append(c.Embeddings, recordEmbeddings)
		}
	}

	return nil
}

func (c *CollectionAddOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionAddOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionAddOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionAddOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionAddOp) Operation() OperationType {
	return OperationCreate
}

type CollectionAddOption func(update *CollectionAddOp) error

func WithTexts(documents ...string) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		if len(documents) == 0 {
			return errors.New("at least one document is required")
		}
		if update.Documents == nil {
			update.Documents = make([]Document, 0)
		}
		for _, text := range documents {
			update.Documents = append(update.Documents, NewTextDocument(text))
		}
		return nil
	}
}

func WithMetadatas(metadatas ...DocumentMetadata) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.Metadatas = metadatas
		return nil
	}
}

func WithIDs(ids ...DocumentID) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		for _, id := range ids {
			update.Ids = append(update.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithIDGenerator(idGenerator IDGenerator) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.IDGenerator = idGenerator
		return nil
	}
}

func WithEmbeddings(embeddings ...embeddings.Embedding) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.Embeddings = embeddings
		return nil
	}
}

// Update

type CollectionUpdateOp struct {
	Ids        []DocumentID           `json:"ids"`
	Documents  []Document             `json:"documents,omitempty"`
	Metadatas  []DocumentMetadata     `json:"metadatas,omitempty"`
	Embeddings []embeddings.Embedding `json:"embeddings"`
	Records    []Record               `json:"-"`
}

func NewCollectionUpdateOp(opts ...CollectionUpdateOption) (*CollectionUpdateOp, error) {
	update := &CollectionUpdateOp{}
	for _, opt := range opts {
		err := opt(update)
		if err != nil {
			return nil, err
		}
	}
	return update, nil
}

func (c *CollectionUpdateOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	// invariants:
	// documents only - we embed
	// documents + embeddings - we skip
	// embeddings only - we skip
	if len(c.Documents) > 0 && len(c.Embeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		texts := make([]string, len(c.Documents))
		for i, doc := range c.Documents {
			texts[i] = doc.ContentString()
		}
		embeddings, err := ef.EmbedDocuments(ctx, texts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		for i, embedding := range embeddings {
			if i >= len(c.Embeddings) {
				c.Embeddings = append(c.Embeddings, embedding)
			} else {
				c.Embeddings[i] = embedding
			}
		}
	}
	return nil
}

func (c *CollectionUpdateOp) PrepareAndValidate() error {
	// invariants
	// - at least one ID or one record is required
	// - if IDs are provided, they must be unique
	// - if IDs are provided, the number of documents or embeddings must match the number of IDs
	// - if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Ids) == 0 && len(c.Records) == 0 {
		return errors.New("at least one ID or record is required.") // TODO add link to docs
	}

	// if IDs are provided, they must be unique
	idSet := make(map[DocumentID]struct{})
	for _, id := range c.Ids {
		if _, exists := idSet[id]; exists {
			return errors.Errorf("duplicate id found: %s", id)
		}
		idSet[id] = struct{}{}
	}

	// if IDs are provided, the number of documents or embeddings must match the number of IDs
	if len(c.Documents) > 0 && len(c.Ids) != len(c.Documents) {
		return errors.Errorf("documents (%d) must match the number of ids (%d)", len(c.Documents), len(c.Ids))
	}

	if len(c.Embeddings) > 0 && len(c.Ids) != len(c.Embeddings) {
		return errors.Errorf("embeddings (%d) must match the number of ids (%d)", len(c.Embeddings), len(c.Ids))
	}

	// if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Metadatas) > 0 && len(c.Ids) != len(c.Metadatas) {
		return errors.Errorf("metadatas (%d) must match the number of ids (%d)", len(c.Metadatas), len(c.Ids))
	}

	if len(c.Records) > 0 {
		for _, record := range c.Records {
			err := record.Validate()
			if err != nil {
				return errors.Wrap(err, "record validation failed")
			}
			recordIds, recordDocuments, recordEmbeddings, recordMetadata := record.Unwrap()
			c.Ids = append(c.Ids, recordIds)
			c.Documents = append(c.Documents, recordDocuments)
			c.Metadatas = append(c.Metadatas, recordMetadata)
			c.Embeddings = append(c.Embeddings, recordEmbeddings)
		}
	}

	return nil
}

func (c *CollectionUpdateOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionUpdateOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionUpdateOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionUpdateOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionUpdateOp) Operation() OperationType {
	return OperationUpdate
}

type CollectionUpdateOption func(update *CollectionUpdateOp) error

func WithTextsUpdate(documents ...string) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		if len(documents) == 0 {
			return errors.New("at least one document is required")
		}
		if update.Documents == nil {
			update.Documents = make([]Document, 0)
		}
		for _, text := range documents {
			update.Documents = append(update.Documents, NewTextDocument(text))
		}
		return nil
	}
}

func WithMetadatasUpdate(metadatas ...DocumentMetadata) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		update.Metadatas = metadatas
		return nil
	}
}

func WithIDsUpdate(ids ...DocumentID) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		for _, id := range ids {
			update.Ids = append(update.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithEmbeddingsUpdate(embeddings ...embeddings.Embedding) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		update.Embeddings = embeddings
		return nil
	}
}

// Delete

type CollectionDeleteOp struct {
	FilterOp
	FilterIDOp
}

func NewCollectionDeleteOp(opts ...CollectionDeleteOption) (*CollectionDeleteOp, error) {
	del := &CollectionDeleteOp{}
	for _, opt := range opts {
		err := opt(del)
		if err != nil {
			return nil, err
		}
	}
	return del, nil
}

func (c *CollectionDeleteOp) PrepareAndValidate() error {
	if len(c.Ids) == 0 && c.Where == nil && c.WhereDocument == nil {
		return errors.New("at least one filter is required, ids, where or whereDocument")
	}

	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return err
		}
	}

	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *CollectionDeleteOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionDeleteOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionDeleteOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionDeleteOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionDeleteOp) Operation() OperationType {
	return OperationDelete
}

type CollectionDeleteOption func(update *CollectionDeleteOp) error

func WithWhereDelete(where WhereFilter) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		delete.Where = where
		return nil
	}
}

func WithWhereDocumentDelete(whereDocument WhereDocumentFilter) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		delete.WhereDocument = whereDocument
		return nil
	}
}

func WithIDsDelete(ids ...DocumentID) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		for _, id := range ids {
			delete.Ids = append(delete.Ids, DocumentID(id))
		}
		return nil
	}
}

type CollectionConfiguration interface {
	GetRaw(key string) (interface{}, bool)
}
