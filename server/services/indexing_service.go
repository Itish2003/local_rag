package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/fsnotify/fsnotify"
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
	chunks, err := splitter.SplitText(string(content))
	if err != nil {
		return err
	}
	log.Printf("INDEXER: Split %s into %d chunks.", path, len(chunks))

	for i, chunk := range chunks {
		embeddingVector, err := s.ragService.EmbedTextWithOllama(ctx, chunk)
		if err != nil {
			return fmt.Errorf("could not embed chunk %d of %s: %w", i, path, err)
		}
		embedding := embeddings.NewEmbeddingFromFloat32(embeddingVector)
		metadata := chromago.NewDocumentMetadata(
			chromago.NewStringAttribute("source_file", path),
			chromago.NewStringAttribute("file_hash", hash),
			chromago.NewIntAttribute("chunk_num", int64(i)),
		)
		docID := chromago.DocumentID(fmt.Sprintf("%s-chunk%d", uuid.New().String(), i))
		err = s.collection.Add(ctx,
			chromago.WithIDs(docID),
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
	results, err := s.collection.Get(ctx)
	if err != nil {
		return nil, err
	}
	metadatas := results.GetMetadatas()
	for _, meta := range metadatas {
		if meta != nil {
			// Try to marshal and unmarshal to map[string]interface{}
			jsonBytes, err := json.Marshal(meta)
			if err != nil {
				continue
			}
			var metaMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &metaMap); err != nil {
				continue
			}
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
	// Use the EqString helper to build a WhereClause for source_file == path
	where := chromago.EqString("source_file", path)
	return s.collection.Delete(ctx, chromago.WithWhereDelete(where))
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
}
