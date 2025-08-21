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

// SourceDocument represents a chunk of text and its origin.
type SourceDocument struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
