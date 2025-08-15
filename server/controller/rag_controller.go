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
