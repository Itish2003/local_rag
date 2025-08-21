Of course. Let's move forward and implement the exciting new features that will elevate your project to the next level. This guide will walk you through implementing each of the four key enhancements we discussed, using best practices and information grounded in current library documentation.

Here is the step-by-step implementation guide.

### Enhancement 1: Live File Watching with `fsnotify`

**Goal:** Automatically update the index in real-time when files are created, modified, or deleted, without needing to restart the server.

**Why:** This makes the RAG system truly dynamic. As you work on your notes, the knowledge base is always in sync, providing the most up-to-date context for your queries.

#### **Step 1: Install `fsnotify` Package**

First, add the library to your project.

```bash
cd server
go get github.com/fsnotify/fsnotify
go mod tidy
```

#### **Step 2: Enhance the `FileIndexingService`**

We will add a new method, `WatchDirectory`, to the `indexing_service.go` file. It will create and manage the file watcher. Note that `fsnotify` watches directories, not individual files, which is a more robust approach.

**Edit `server/services/indexing_service.go`:**

```go
package services

import (
	// Add fsnotify to your imports
	"github.com/fsnotify/fsnotify"
	// ... other imports are the same
)
// ... FileIndexingService struct and NewFileIndexingService are the same ...

// WatchDirectory starts a long-running process to watch for file changes in real-time.
func (s *FileIndexingService) WatchDirectory(ctx context.Context, dirPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("WATCHER ERROR: Failed to create file watcher: %v", err)
		return
	}
	defer watcher.Close()

	// Goroutine to handle events from the watcher.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// We only care about supported file types.
				if !isSupportedFile(event.Name) {
					continue
				}

				log.Printf("WATCHER EVENT: %s", event)

				// A Create or Write event means we need to index the file.
				// Many editors perform a "write" by creating a temp file and renaming,
				// which can trigger multiple events. We handle Create and Write the same.
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Printf("WATCHER: File modified/created: %s. Re-indexing...", event.Name)
					hash, err := calculateFileHash(event.Name)
					if err != nil {
						log.Printf("WATCHER WARN: Could not hash file %s: %v", event.Name, err)
						continue
					}
					// Delete old versions before re-indexing
					s.deleteDocumentsByFilepath(ctx, event.Name)
					if err := s.processAndEmbedFile(ctx, event.Name, hash); err != nil {
						log.Printf("WATCHER ERROR: Failed to process file %s: %v", event.Name, err)
					}
				} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					// Rename is often treated as Remove by watchers.
					log.Printf("WATCHER: File removed/renamed: %s. Removing from index...", event.Name)
					if err := s.deleteDocumentsByFilepath(ctx, event.Name); err != nil {
						log.Printf("WATCHER ERROR: Failed to delete records for %s: %v", event.Name, err)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("WATCHER ERROR: %v", err)
			case <-ctx.Done():
				log.Println("WATCHER: Context cancelled, shutting down watcher.")
				return
			}
		}
	}()

	log.Printf("WATCHER: Watching directory: %s", dirPath)
	err = watcher.Add(dirPath)
	if err != nil {
		log.Printf("WATCHER ERROR: Failed to add path to watcher: %v", err)
	}

	// Block until the context is cancelled (e.g., server shutdown).
	<-ctx.Done()
}


// ... (rest of the file is the same)
```

#### **Step 3: Update `main.go` to Start the Watcher**

Modify `main.go` to launch the `WatchDirectory` method in a new goroutine, just like the initial scan.

**Edit `server/main.go`:**

```go
// ... (imports)

func main() {
    // ... (godotenv.Load() and other setup)

    // ==========================================================
	// ===== NEW: Instantiate and run the Indexing Service ======
	// ==========================================================
	indexingService := services.NewFileIndexingService(collection, ragService)
	go func() {
		indexPath := os.Getenv("INDEX_PATH")
		if indexPath == "" {
			log.Println("WARN: INDEX_PATH not set in .env. File indexing will not run.")
			return
		}
		absPath, err := filepath.Abs(indexPath)
		if err != nil {
			log.Printf("ERROR: Invalid INDEX_PATH: %v", err)
			return
		}
		
        // --- This part is new ---
		// Create a cancellable context for the watcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancellation on main exit

		// Run the initial scan
		indexingService.ScanAndIndexDirectory(ctx, absPath)

		// Start the real-time watcher in a new goroutine
		go indexingService.WatchDirectory(ctx, absPath)
        // --- End of new part ---
	}()

    // ... (rest of main function)
}

// ... (getOrCreateCollectionV2 function is unchanged) ...
```

#### **Step 4: Test the Feature**

1.  Run your entire application stack (`chroma`, `ollama`, `go run main.go`).
2.  Observe the initial scan logs.
3.  **While the server is running**, go to your `notes/` directory.
4.  **Create a new file** (`test_live.md`) and add some text. Watch the server logs – you should see the watcher detect the `CREATE` and `WRITE` events and index the new file.
5.  **Modify the file** `project_ideas.md`. Watch the logs for the `WRITE` event and re-indexing message.
6.  **Delete the file** `test_live.md`. Watch the logs for the `REMOVE` event and the deletion message.

---

### Enhancement 2: Display Document Sources on Frontend

**Goal:** Show the user the source file for the information used to generate an answer.

**Why:** This increases transparency and trust. Users can see *why* the AI gave a certain answer and can refer back to the original document for more context.

#### **Step 1: Modify Backend to Return Source Metadata**

We need to change the `retrieveDocuments` function to return the full document object (text + metadata) and then update the response model.

**A. Create a new `SourceDocument` struct in `server/models/notes.go`:**

```go
package models

// ... (existing Note and GetAllNotesResponse structs)

// SourceDocument represents a chunk of text and its origin.
type SourceDocument struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}
```

**B. Update the `QueryRAGResponse` in `server/models/response.go`:**

```go
package models

// ... (InjestDataResponse struct is the same)

type QueryRAGResponse struct {
	Answer     string           `json:"answer"`
	SourceDocs []SourceDocument `json:"source_docs,omitempty"` // <-- CHANGE: Use the new struct
	Error      string           `json:"error,omitempty"`
}
```

**C. Update the `RAGService` in `server/services/rag_service.go`:**

We'll modify `retrieveDocuments` to pass the full document information up to `QueryRAG`.

```go
// ... (imports)

// RAGService interface defines methods for RAG operations
type RAGService interface {
	// ... (no changes to the interface)
}

// ... (ragServiceImpl struct is the same)
// ... (GetAllNotes and IngestNote are the same)

// QueryRAG implements RAGService
func (r *ragServiceImpl) QueryRAG(c context.Context, req models.QueryTextRequest) (*models.QueryRAGResponse, error) {
	log.Printf("SERVICE: Querying RAG with: '%s'", req.Query)

	// --- CHANGE: This function now returns []models.SourceDocument ---
	retrievedDocs, err := r.retrieveDocuments(c, req.Query, 3)
	if err != nil {
		return nil, err
	}
	// --- END CHANGE ---

	// Extract just the text for the prompt context
	var docTexts []string
	for _, doc := range retrievedDocs {
		docTexts = append(docTexts, doc.Text)
	}
	ragPrompt := r.createRAGPrompt(req.Query, docTexts)

	geminiAnswer, err := r.generateResponseWithGemini(c, ragPrompt)
	if err != nil {
		return nil, fmt.Errorf("could not generate response from gemini: %w", err)
	}

	response := &models.QueryRAGResponse{
		Answer:     geminiAnswer,
		SourceDocs: retrievedDocs, // <-- CHANGE: Pass the full source documents
	}
	return response, nil
}

// retrieveDocuments queries ChromaDB and now returns []models.SourceDocument
func (r *ragServiceImpl) retrieveDocuments(c context.Context, query string, nResults int) ([]models.SourceDocument, error) { // <-- CHANGE: Return type
	log.Printf("SERVICE-HELPER: Retrieving documents from ChromaDB...")

	queryEmbedding, err := r.EmbedText(c, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query text: %w", err)
	}
	embedding := embeddings.NewEmbeddingFromFloat32(queryEmbedding)

	// --- CHANGE: We now need to include metadata in the results ---
	results, err := r.collection.Query(
		c,
		chromago.WithQueryEmbeddings(embedding),
		chromago.WithNResults(nResults),
		chromago.WithInclude(chromago.Document, chromago.Metadata), // <-- Explicitly include metadata
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chromadb: %w", err)
	}

	// --- CHANGE: Process results into the new struct ---
	var documents []models.SourceDocument
	documentGroups := results.GetDocumentsGroups()
	metadataGroups := results.GetMetadatasGroups()

	if len(documentGroups) > 0 {
		for i, doc := range documentGroups[0] {
			if doc.ContentString() != "" {
				sourceDoc := models.SourceDocument{
					Text:     doc.ContentString(),
					Metadata: metadataGroups[0][i].GetValues(),
				}
				documents = append(documents, sourceDoc)
			}
		}
	}
	// --- END CHANGE ---

	log.Printf("SERVICE-HELPER: Retrieved %d documents", len(documents))
	return documents, nil
}


// ... (rest of the file is the same)
```

#### **Step 2: Update the Frontend `QueryResults` Component**

Now, we'll modify the React component to display the richer source information.

**Edit `client/src/components/QueryResults.jsx`:**

```jsx
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Accordion from '@mui/material/Accordion';
import AccordionSummary from '@mui/material/AccordionSummary';
import AccordionDetails from '@mui/material/AccordionDetails';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Fragment } from 'react';

function QueryResults({ result, loading, error }) {
  if (loading) return <Typography>Loading...</Typography>;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!result) return null;

  return (
    <Card sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 1 }}>Answer</Typography>
        <Typography sx={{ mb: 3, whiteSpace: 'pre-wrap' }}>{result.answer || 'No answer.'}</Typography>

        {result.source_docs && result.source_docs.length > 0 && (
          <Box>
            <Typography variant="subtitle1" sx={{ mb: 1 }}>Sources</Typography>
            {result.source_docs.map((doc, idx) => (
              <Accordion key={idx} sx={{ '&:before': { display: 'none' }, boxShadow: 'none', border: '1px solid rgba(0, 0, 0, 0.12)' }}>
                <AccordionSummary
                  expandIcon={<ExpandMoreIcon />}
                  aria-controls={`panel${idx}-content`}
                  id={`panel${idx}-header`}
                >
                  <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                    {/* Display filename from metadata */}
                    Source {idx + 1}: {doc.metadata?.source_file?.split('/').pop() || 'User Note'}
                  </Typography>
                </AccordionSummary>
                <AccordionDetails sx={{ backgroundColor: 'rgba(0, 0, 0, 0.03)' }}>
                  <Typography variant="caption" display="block" color="text.secondary">
                    Chunk {doc.metadata?.chunk_num !== undefined ? doc.metadata.chunk_num + 1 : 'N/A'}
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 1 }}>
                    {doc.text}
                  </Typography>
                </AccordionDetails>
              </Accordion>
            ))}
          </Box>
        )}
      </CardContent>
    </Card>
  );
}

export default QueryResults;
```

#### **Step 3: Test the Feature**

1.  Restart your Go server to apply the backend changes.
2.  Refresh your React frontend.
3.  Ask a question related to one of your indexed files (e.g., "What is the AI knowledge base project?").
4.  The answer should appear, followed by a new "Sources" section with expandable accordions showing the filename, chunk number, and the exact text used to generate the answer.

---

### Enhancement 3: Support for PDF Files

**Goal:** Allow the indexer to read and process text from PDF files.

**Why:** A vast amount of knowledge is stored in PDFs. Supporting them dramatically increases the utility of your RAG system. We will use the powerful `unidoc` library for this.

#### **Step 1: Install `unipdf` Package**

`unipdf` is a commercial library, but it offers a free metered license that is suitable for development and many production use cases.

1.  **Sign up** for a free metered key at [unidoc.io](https://cloud.unidoc.io).
2.  **Add the library** to your project:
    ```bash
    cd server
    go get github.com/unidoc/unipdf/v3
    go mod tidy
    ```
3.  **Add your license key** to `server/.env`:
    ```env
    # ... other keys
    UNIDOC_LICENSE_KEY="your-api-key-from-unidoc"
    ```

#### **Step 2: Create a Text Extractor Service**

To keep the logic clean, we'll create a new file responsible for extracting text from different file types.

**Create a new file `server/services/extractor_service.go`:**

```go
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func init() {
	// Make sure to load your metered license key prior to using the library.
	// This is just one of the ways to load your license key.
	err := license.SetMeteredKey(os.Getenv("UNIDOC_LICENSE_KEY"))
	if err != nil {
		fmt.Printf("ERROR: Failed to set Unidoc license key: %v. PDF processing will fail.\n", err)
	}
}

// ExtractTextFromFile reads a file and returns its text content.
// It automatically handles different file types.
func ExtractTextFromFile(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".md":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case ".pdf":
		return extractTextFromPDF(path)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// extractTextFromPDF uses UniPDF to get all text from a PDF file.
func extractTextFromPDF(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return "", err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			return "", err
		}

		ex, err := extractor.New(page)
		if err != nil {
			return "", err
		}

		text, err := ex.ExtractText()
		if err != nil {
			return "", err
		}
		sb.WriteString(text)
		sb.WriteString("\n\n") // Add space between pages
	}

	return sb.String(), nil
}

```

#### **Step 3: Update `FileIndexingService` to Use the Extractor**

Now, we'll modify the indexer to use our new `ExtractTextFromFile` function.

**Edit `server/services/indexing_service.go`:**

```go
// ... (imports)

// ... (FileIndexingService struct and other methods)

func (s *FileIndexingService) processAndEmbedFile(ctx context.Context, path, hash string) error {
	// --- CHANGE: Use the new extractor service ---
	content, err := ExtractTextFromFile(path)
	if err != nil {
		return fmt.Errorf("could not extract text from %s: %w", path, err)
	}
	// --- END CHANGE ---

	splitter := textsplitter.NewRecursiveCharacter(textsplitter.WithChunkSize(1000), textsplitter.WithChunkOverlap(100))
	chunks, err := splitter.SplitText(ctx, content) // Pass the extracted content
	if err != nil {
		return err
	}
	log.Printf("INDEXER: Split %s into %d chunks.", path, len(chunks))

	// ... (rest of the function is the same)
}


// ... (other functions)

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".pdf": // <-- CHANGE: Add .pdf
		return true
	default:
		return false
	}
}

// ... (rest of the file is the same)
```

#### **Step 4: Test the Feature**

1.  Make sure you have added your `UNIDOC_LICENSE_KEY` to the `.env` file.
2.  Add a PDF file to your `notes/` directory.
3.  Restart your Go server.
4.  Observe the logs. You should see the indexer pick up the new PDF, extract its content, and create chunks.
5.  Use the frontend or `curl` to ask a question specifically about the content of the PDF. The system should answer correctly and cite the PDF as a source.

---

### Enhancement 4: Indexer Status API Endpoint

**Goal:** Create an API endpoint to check the status of the indexing service.

**Why:** This is crucial for observability. It allows a UI (or an administrator) to understand what the indexer is doing, how many files are indexed, and if it's currently busy.

#### **Step 1: Add a Status Tracker to `FileIndexingService`**

We'll add fields to track the status and use mutexes to ensure thread-safe access, as the watcher and the API will access this data concurrently.

**Edit `server/services/indexing_service.go`:**

```go
import (
	"sync" // <-- Add sync for mutex
	// ... other imports
)

// IndexerStatus represents the current state of the indexer.
type IndexerStatus struct {
	IsScanning  bool   `json:"is_scanning"`
	CurrentFile string `json:"current_file"`
	FileCount   int    `json:"file_count"`
}

// FileIndexingService handles scanning, chunking, and embedding files.
type FileIndexingService struct {
	collection chromago.Collection
	ragService RAGService
	// --- NEW: Status tracking fields ---
	mu     sync.RWMutex
	status IndexerStatus
	// --- END NEW ---
}

// NewFileIndexingService creates a new indexing service.
func NewFileIndexingService(collection chromago.Collection, ragService RAGService) *FileIndexingService {
	return &FileIndexingService{
		collection: collection,
		ragService: ragService,
		status:     IndexerStatus{IsScanning: false}, // Initial state
	}
}

// --- NEW: Helper methods to update status safely ---
func (s *FileIndexingService) setStatus(scanning bool, currentFile string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.IsScanning = scanning
	s.status.CurrentFile = currentFile
}

func (s *FileIndexingService) setFileCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.FileCount = count
}

// GetStatus returns the current status of the indexer.
func (s *FileIndexingService) GetStatus() IndexerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}
// --- END NEW ---

// ScanAndIndexDirectory - update to use status methods
func (s *FileIndexingService) ScanAndIndexDirectory(ctx context.Context, dirPath string) {
	log.Printf("INDEXER: Starting directory scan for: %s", dirPath)
	s.setStatus(true, "Starting initial scan...") // <-- Set status

	// ... (your existing logic)
    
	// After getting the index state:
	s.setFileCount(len(indexedFiles))

	// ...
	// Inside filepath.Walk, before processing a file:
	s.setStatus(true, path)
	// ...

	log.Println("INDEXER: Directory scan finished.")
	s.setStatus(false, "") // <-- Reset status when done
}

// ... (Update WatchDirectory similarly to set status on events)
func (s *FileIndexingService) WatchDirectory(ctx context.Context, dirPath string) {
    // ...
    // Inside the event loop for a write/create event:
    s.setStatus(true, event.Name)
    // ... process file ...
    s.setStatus(false, "")
    s.setFileCount(s.status.FileCount + 1) // Or decrement on delete

    // Inside the event loop for a remove event:
    s.setStatus(true, "Deleting "+event.Name)
    // ... delete file ...
    s.setStatus(false, "")
    s.setFileCount(s.status.FileCount - 1)
    // ...
}
```

#### **Step 2: Create a New Controller and Route**

**A. Add a new `IndexingController` in `server/controller/indexing_controller.go`:**

```go
package controller

import (
	"github/itish2003/rag/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type IndexingController struct {
	indexingService *services.FileIndexingService
}

func NewIndexingController(service *services.FileIndexingService) *IndexingController {
	return &IndexingController{
		indexingService: service,
	}
}

func (c *IndexingController) GetStatus(ctx *gin.Context) {
	status := c.indexingService.GetStatus()
	ctx.JSON(http.StatusOK, status)
}
```

**B. Update `main.go` to add the new route:**

```go
// in main.go
func main() {
    // ...
	// After creating ragController
	indexingController := controller.NewIndexingController(indexingService)

	// API routes
	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/notes", ragController.IngestNote)
		apiV1.GET("/notes", ragController.GetAllNotes)
		apiV1.POST("/query", ragController.QueryRAG)
        // --- NEW ROUTE ---
		apiV1.GET("/indexer/status", indexingController.GetStatus)
	}
    // ...
}
```

#### **Step 3: Test the Endpoint**

1.  Restart the Go server.
2.  While the initial scan is running, open a new terminal and run:
    ```bash
    curl http://localhost:8080/api/v1/indexer/status
    ```
3.  You should see a JSON response indicating that `is_scanning` is `true` and showing the file it's currently processing.
4.  Once the scan is complete, run the command again. `is_scanning` should be `false`.

You have now successfully implemented all four major enhancements, making your project significantly more powerful, dynamic, and user-friendly.