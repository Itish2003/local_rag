package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileActions handles the actual file system operations.
type FileActions struct {
	NotesDir string // The absolute path to the notes directory
}

func NewFileActions() (*FileActions, error) {
	notesPath := os.Getenv("INDEX_PATH")
	if notesPath == "" {
		return nil, fmt.Errorf("INDEX_PATH environment variable not set")
	}
	absPath, err := filepath.Abs(notesPath)
	if err != nil {
		return nil, fmt.Errorf("could not determine absolute path for INDEX_PATH: %w", err)
	}
	return &FileActions{NotesDir: absPath}, nil
}

// sanitizeFilename ensures the filename is safe and within the notes directory.
func (fa *FileActions) sanitizeFilename(filename string) (string, error) {
	if !strings.HasSuffix(filename, ".md") {
		return "", fmt.Errorf("filename must end with .md")
	}
	// This prevents path traversal attacks (e.g., filename = "../../../etc/passwd")
	cleanPath := filepath.Join(fa.NotesDir, filepath.Base(filename))
	if !strings.HasPrefix(cleanPath, fa.NotesDir) {
		return "", fmt.Errorf("invalid filename, attempts to escape notes directory")
	}
	return cleanPath, nil
}

func (fa *FileActions) CreateMarkdownFile(filename, content string) string {
	path, err := fa.sanitizeFilename(filename)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Sprintf("Error: File '%s' already exists.", filename)
	}
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to create file '%s': %v", filename, err)
	}
	return fmt.Sprintf("Success: File '%s' created.", filename)
}

func (fa *FileActions) DeleteMarkdownFile(filename string) string {
	path, err := fa.sanitizeFilename(filename)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	err = os.Remove(path)
	if err != nil {
		return fmt.Sprintf("Error: Failed to delete file '%s': %v", filename, err)
	}
	return fmt.Sprintf("Success: File '%s' deleted.", filename)
}

func (fa *FileActions) EditMarkdownFile(filename, content string) string {
	path, err := fa.sanitizeFilename(filename)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to open file '%s' for editing: %v", filename, err)
	}
	defer f.Close()

	if _, err = f.WriteString("\n\n" + content); err != nil {
		return fmt.Sprintf("Error: Failed to write to file '%s': %v", filename, err)
	}
	return fmt.Sprintf("Success: Content appended to file '%s'.", filename)
}
