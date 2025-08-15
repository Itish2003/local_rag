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
