package models

type IngestDataRequest struct {
	Text string `json:"text"`
}

type QueryTextRequest struct {
	Query     string `json:"query"`
	SessionID string `json:"sessionID,omitempty"`
}
