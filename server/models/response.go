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
