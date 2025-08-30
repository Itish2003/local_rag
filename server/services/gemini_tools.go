package services

import "google.golang.org/genai"

// GetAllTools defines the list of functions available to Gemini for file manipulation.
func GetAllTools() []*genai.Tool {
	return []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "retrieveDocuments",
					Description: "Search the user's notes for documents relevant to a specific topic or question.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"query": {
								Type:        genai.TypeString,
								Description: "The specific topic or question to search for in the document store. This should be a concise search query.",
							},
						},
						Required: []string{"query"},
					},
				},
				{
					Name:        "createMarkdownFile",
					Description: "Create a new markdown file with specified content in the notes directory.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"filename": {
								Type:        genai.TypeString,
								Description: "The name of the file to create, e.g., 'my_thoughts.md'. Must end with .md",
							},
							"content": {
								Type:        genai.TypeString,
								Description: "The markdown content to write into the file.",
							},
						},
						Required: []string{"filename", "content"},
					},
				},
				{
					Name:        "deleteMarkdownFile",
					Description: "Delete a markdown file from the notes directory.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"filename": {
								Type:        genai.TypeString,
								Description: "The name of the file to delete, e.g., 'old_note.md'.",
							},
						},
						Required: []string{"filename"},
					},
				},
				{
					Name:        "editMarkdownFile",
					Description: "Append new content to an existing markdown file in the notes directory.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"filename": {
								Type:        genai.TypeString,
								Description: "The name of the file to edit, e.g., 'project_ideas.md'.",
							},
							"content": {
								Type:        genai.TypeString,
								Description: "The new content to append to the end of the file.",
							},
						},
						Required: []string{"filename", "content"},
					},
				},
			},
		},
	}
}
