package models

type InjestDataResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type QueryRAGResponse struct {
	Answer     string           `json:"answer"`
	SourceDocs []SourceDocument `json:"source_docs,omitempty"`
	Error      string           `json:"error,omitempty"`
	SessionID  string           `json:"sessionID"`
}
