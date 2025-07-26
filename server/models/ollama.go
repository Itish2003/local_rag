package models

// OllamaEmbedRequest is used to structure the request to the Ollama embedding API.
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse is used to parse the embedding from the Ollama API response.
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}
