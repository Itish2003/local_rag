Of course. Here is a detailed, step-by-step guide to implement the automated file indexing system into your existing project.

This guide will walk you through:
1.  Installing necessary packages.
2.  Updating the configuration.
3.  Refactoring the `RAGService` to make the embedding function reusable.
4.  Creating the new `FileIndexingService`.
5.  Integrating the new service into `main.go` to run on startup.
6.  Testing the entire workflow.

---

### Step 1: Install New Go Packages

First, we need to add two new dependencies to your Go project for loading the `.env` file and for text splitting.

Navigate to your `server` directory in the terminal and run the following commands:

```bash
cd server
go get github.com/joho/godotenv
go get github.com/tmc/langchaingo
go mod tidy
```

### Step 2: Update Your Configuration

The system needs to know which directory to index. We'll add this path to the `.env` file.

**Edit `server/.env` and add the new `INDEX_PATH` variable at the bottom:**

```env
GEMINI_API_KEY='AIzaSyDb7JayvdzIk6GtSJ1V-ktcQw0qoBww3K4'
OLLAMA_URL=http://localhost:11434/api/embeddings
OLLAMA_MODEL=nomic-embed-text
GEMINI_MODEL=gemini-1.5-flash
RETRIEVAL_N_RESULTS=3

# Add this new line. This path is relative to where the Go server runs.
# In our case, it points to the 'notes' directory at the project root.
INDEX_PATH='../notes'
```

### Step 3: Refactor `RAGService` for Reusability

To avoid duplicating code, we will expose the embedding logic from `ragServiceImpl` so our new `FileIndexingService` can use it.

**Modify `server/services/rag_service.go`:**

1.  **Add `EmbedText` to the `RAGService` interface.**
2.  **Rename `embedTextWithOllama` to `EmbedText`** to satisfy the interface.
3.  **Update internal calls** to use the new public `EmbedText` method.

Here is the updated file. Changes are marked with `// <-- CHANGE`.

```go
// =====================================================
// server/services/rag_service.go
package services

import (
	// ... existing imports
)

// RAGService interface defines methods for RAG operations
type RAGService interface {
	IngestNote(c context.Context, req models.IngestDataRequest) error
	QueryRAG(c context.Context, req models.QueryTextRequest) (*models.QueryRAGResponse, error)
	GetAllNotes(c context.Context) (*models.GetAllNotesResponse, error)
	EmbedText(c context.Context, textToEmbed string) ([]float32, error) // <-- CHANGE: Add this method
}

// ... (ragServiceImpl struct is unchanged) ...

// ... (GetAllNotes function is unchanged) ...

// IngestNote implements RAGService
func (r *ragServiceImpl) IngestNote(c context.Context, req models.IngestDataRequest) error {
	log.Printf("SERVICE: Ingesting note: '%s'", req.Text)

	embeddingVector, err := r.EmbedText(c, req.Text) // <-- CHANGE: Use new public method
	if err != nil {
		return fmt.Errorf("could not generate embedding for note: %w", err)
	}

    // ... (rest of the function is unchanged) ...
}

// ... (QueryRAG function is unchanged) ...

// retrieveDocuments queries ChromaDB for similar documents using v2 API
func (r *ragServiceImpl) retrieveDocuments(c context.Context, query string, nResults int) ([]string, error) {
	log.Printf("SERVICE-HELPER: Retrieving documents from ChromaDB using v2 API...")

	// 1. Embed the query text using Ollama
	queryEmbedding, err := r.EmbedText(c, query) // <-- CHANGE: Use new public method
	if err != nil {
		return nil, fmt.Errorf("failed to embed query text: %w", err)
	}
    // ... (rest of the function is unchanged) ...
}

// ... (generateResponseWithGemini and createRAGPrompt are unchanged) ...

// EmbedText generates embeddings using Ollama.
// This function is now public to be used by other services.
func (r *ragServiceImpl) EmbedText(c context.Context, textToEmbed string) ([]float32, error) { // <-- CHANGE: Renamed from embedTextWithOllama
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


// ... (NewRAGService function is unchanged) ...

```

### Step 4: Create the New `FileIndexingService`

This is the core of the new functionality. Create a new file that will contain all the logic for scanning the directory, hashing files, chunking text, and interacting with ChromaDB.

**Create a new file: `server/services/indexing_service.go`**

```go
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/textsplitter"
)

// FileIndexingService handles scanning, chunking, and embedding files.
type FileIndexingService struct {
	collection chromago.Collection
	ragService RAGService
}

// NewFileIndexingService creates a new indexing service.
func NewFileIndexingService(collection chromago.Collection, ragService RAGService) *FileIndexingService {
	return &FileIndexingService{
		collection: collection,
		ragService: ragService,
	}
}

// IndexState holds the current hash of a file in our index.
type IndexState struct {
	Hash string
}

// ScanAndIndexDirectory is the main function to sync the directory with ChromaDB.
func (s *FileIndexingService) ScanAndIndexDirectory(ctx context.Context, dirPath string) {
	log.Printf("INDEXER: Starting directory scan for: %s", dirPath)

	indexedFiles, err := s.getCurrentIndexState(ctx)
	if err != nil {
		log.Printf("INDEXER ERROR: Could not get current index state: %v", err)
		return
	}
	log.Printf("INDEXER: Found %d files currently in the index.", len(indexedFiles))

	localFiles := make(map[string]bool)
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isSupportedFile(path) {
			localFiles[path] = true
			hash, err := calculateFileHash(path)
			if err != nil {
				log.Printf("INDEXER WARN: Could not hash file %s: %v", path, err)
				return nil
			}

			if state, ok := indexedFiles[path]; ok {
				if state.Hash == hash {
					return nil // File is unchanged, skip.
				}
				log.Printf("INDEXER: File has changed: %s. Re-indexing...", path)
				if err := s.deleteDocumentsByFilepath(ctx, path); err != nil {
					log.Printf("INDEXER ERROR: Failed to delete old version of %s: %v", path, err)
					return nil
				}
			}

			log.Printf("INDEXER: Indexing new/modified file: %s", path)
			if err := s.processAndEmbedFile(ctx, path, hash); err != nil {
				log.Printf("INDEXER ERROR: Failed to process file %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("INDEXER ERROR: Error walking the path %s: %v", dirPath, err)
	}

	// Handle deletions
	for path := range indexedFiles {
		if !localFiles[path] {
			log.Printf("INDEXER: File deleted: %s. Removing from index...", path)
			if err := s.deleteDocumentsByFilepath(ctx, path); err != nil {
				log.Printf("INDEXER ERROR: Failed to delete records for %s: %v", path, err)
			}
		}
	}
	log.Println("INDEXER: Directory scan finished.")
}

func (s *FileIndexingService) processAndEmbedFile(ctx context.Context, path, hash string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	splitter := textsplitter.NewRecursiveCharacter(textsplitter.WithChunkSize(1000), textsplitter.WithChunkOverlap(100))
	chunks, err := splitter.SplitText(ctx, string(content))
	if err != nil {
		return err
	}
	log.Printf("INDEXER: Split %s into %d chunks.", path, len(chunks))

	for i, chunk := range chunks {
		embeddingVector, err := s.ragService.EmbedText(ctx, chunk)
		if err != nil {
			return fmt.Errorf("could not embed chunk %d of %s: %w", i, path, err)
		}
		embedding := embeddings.NewEmbeddingFromFloat32(embeddingVector)
		metadata := chromago.NewDocumentMetadata(
			chromago.NewStringAttribute("source_file", path),
			chromago.NewStringAttribute("file_hash", hash),
			chromago.NewIntAttribute("chunk_num", int64(i)),
		)
		docID := fmt.Sprintf("%s-chunk%d", uuid.New().String(), i)
		err = s.collection.Add(ctx,
			chromago.WithIDs(chromago.DocumentID(docID)),
			chromago.WithTexts(chunk),
			chromago.WithEmbeddings(embedding),
			chromago.WithMetadatas(metadata),
		)
		if err != nil {
			return fmt.Errorf("failed to add chunk %d of %s to chromadb: %w", i, path, err)
		}
	}
	return nil
}

func (s *FileIndexingService) getCurrentIndexState(ctx context.Context) (map[string]IndexState, error) {
	state := make(map[string]IndexState)
	results, err := s.collection.Get(ctx, chromago.WithInclude(chromago.Metadata))
	if err != nil {
		return nil, err
	}
	metadatas := results.GetMetadatas()
	for _, meta := range metadatas {
		if meta != nil {
			metaMap := meta.GetValues()
			if path, ok := metaMap["source_file"].(string); ok {
				if hash, ok := metaMap["file_hash"].(string); ok {
					if _, exists := state[path]; !exists {
						state[path] = IndexState{Hash: hash}
					}
				}
			}
		}
	}
	return state, nil
}

func (s *FileIndexingService) deleteDocumentsByFilepath(ctx context.Context, path string) error {
	where := chromago.NewWhere(chromago.NewStringAttribute("source_file", path))
	_, err := s.collection.Delete(ctx, chromago.WithWhere(where))
	return err
}

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md": // Feel free to add more extensions like .go, .py, etc.
		return true
	default:
		return false
	}
}

func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}```

### Step 5: Integrate the Indexer into `main.go`

Now, we'll update the main application entry point to initialize and run our new `FileIndexingService` on startup.

**Modify `server/main.go`:**

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
	"github.com/joho/godotenv" // <-- Add this import
	"google.golang.org/genai"
)

func main() {
	// Load .env file from the current directory
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	// Create HTTP client properly
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// ... (Chroma client and collection setup is unchanged) ...
	chromaClient, err := chromago.NewHTTPClient()
	if err != nil {
		log.Fatalf("FATAL: Failed to create chroma client: %v", err)
	}
	defer func() {
		if errorss := chromaClient.Close(); errorss != nil {
			log.Printf("Warning: Failed to close chroma client: %v", errorss)
		}
	}()
	collection, err := getOrCreateCollectionV2(chromaClient, "test-collection")
	if err != nil {
		log.Fatalf("FATAL: Failed to get or create collection: %v", err)
	}
	// ... (Gemini client setup is unchanged) ...
	geminiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("FATAL: Failed to create Gemini client: %v. Make sure GEMINI_API_KEY is set.", err)
	}
	log.Println("Successfully connected to Google Gemini.")

	// Instantiate services
	ragService := services.NewRAGService(httpClient, collection, geminiClient)
	ragController := controller.NewRAGController(ragService)

	// ==========================================================
	// ===== NEW: Instantiate and run the Indexing Service ======
	// ==========================================================
	indexingService := services.NewFileIndexingService(collection, ragService)
	// Run the indexing in a background goroutine so it doesn't block server start
	go func() {
		indexPath := os.Getenv("INDEX_PATH")
		if indexPath == "" {
			log.Println("WARN: INDEX_PATH not set in .env. File indexing will not run.")
			return
		}
		// Convert to absolute path to be safe
		absPath, err := filepath.Abs(indexPath)
		if err != nil {
			log.Printf("ERROR: Invalid INDEX_PATH: %v", err)
			return
		}
		indexingService.ScanAndIndexDirectory(context.Background(), absPath)
	}()
	// ==========================================================

	// ... (rest of the file with Gin router setup and router.Run is unchanged) ...
	router := gin.Default()

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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "RAG API",
			"version": "1.0.0",
		})
	})

	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/notes", ragController.IngestNote)
		apiV1.GET("/notes", ragController.GetAllNotes)
		apiV1.POST("/query", ragController.QueryRAG)
	}

	port := "8080"
	log.Printf("Go Gin backend server starting on http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}

// ... (getOrCreateCollectionV2 function is unchanged) ...
```

### Step 6: Create Test Files and Run the System

To see the indexer in action, let's create a file in the directory we told it to watch.

1.  **Create a test file:** In the `notes/` directory at the root of your project, create a new file named `project_ideas.md`.
2.  **Add content to the file:**

    ```md
    # Project Ideas for 2025

    ## Q1: AI-Powered Local Knowledge Base
    The main goal is to create a RAG (Retrieval-Augmented Generation) system that runs locally. It should be able to index a directory of markdown and text files. The core technologies will be Go for the backend, Ollama for embeddings, and Gemini for the generative part.

    ## Q2: Personal Finance Dashboard
    A web application to track personal expenses and investments, possibly using Plaid API for bank connections.
    ```

**Now, run the entire application stack:**

1.  **Terminal 1 (ChromaDB):**
    ```bash
    chroma run --path ./my_chroma_data
    ```
2.  **Terminal 2 (Ollama):**
    ```bash
    ollama serve
    # (If you haven't already, pull the model in another terminal)
    # ollama pull nomic-embed-text
    ```
3.  **Terminal 3 (Go Server):**
    ```bash
    cd server
    go run main.go
    ```

**Verify the logs** in Terminal 3. You should see output similar to this:

```
INDEXER: Starting directory scan for: /path/to/your/project/notes
INDEXER: Found 0 files currently in the index.
INDEXER: Indexing new/modified file: ../notes/project_ideas.md
INDEXER: Split ../notes/project_ideas.md into 2 chunks.
... (other server startup logs) ...
INDEXER: Directory scan finished.
```

### Step 7: Test the RAG Pipeline with a Query

Now that your file has been automatically indexed, you can ask a question about its content.

Open a **new terminal** and use `curl` to query your API:

```bash
curl -X POST http://localhost:8080/api/v1/query \
-H "Content-Type: application/json" \
-d '{
  "query": "What are the core technologies for the AI knowledge base project?"
}'
```

You should receive a response from Gemini that correctly identifies Go, Ollama, and Gemini, based on the content of your `project_ideas.md` file.

Congratulations! You have successfully implemented an automated file indexing system for your local RAG project. It will now intelligently keep your knowledge base in sync with your local files every time you start it.