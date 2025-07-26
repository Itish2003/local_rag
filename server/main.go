package main

import (
	"context"
	"github/itish2003/rag/controller"
	"github/itish2003/rag/services"
	"log"
	"net/http"
	"os"
	"time"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

func main() {
	// Create HTTP client properly
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Chroma client
	chromaClient, err := chromago.NewClient(
		chromago.WithBasePath("http://localhost:8000"),
		chromago.WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to create chroma client: %v", err)
	}

	// Create collection
	collection, err := chromaClient.CreateCollection(
		context.Background(),
		"test-collection",
		nil,
		true,
		nil,
		"",
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to create collection: %v", err)
	}

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

	router := gin.Default()
	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/notes", ragController.IngestNote) // Endpoint to create a new note
		apiV1.POST("/query", ragController.QueryRAG)   // Endpoint to ask a question
	}

	// Start the Server
	port := "8080"
	log.Printf("Go Gin backend server starting on http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}
