# Local RAG Server

This directory contains the Go backend server for the Local RAG (Retrieval-Augmented Generation) application. It provides a RESTful API for ingesting notes, querying a vector database, and generating AI responses using Google Gemini. The server also includes a file indexing service that automatically watches a specified directory for changes and keeps the vector database in sync.

## Features

- **RESTful API**: Exposes endpoints for ingesting data, querying the RAG pipeline, and retrieving all notes.
- **Retrieval-Augmented Generation (RAG)**: Combines document retrieval from a ChromaDB vector store with the generative capabilities of Google Gemini.
- **Local File Indexing**: Automatically scans a directory for supported file types (`.txt`, `.md`, `.pdf`), chunks the content, generates embeddings using a local Ollama instance, and stores them in ChromaDB.
- **Real-time File Watching**: Uses a file watcher to detect changes (creations, modifications, deletions) in the indexed directory and updates the vector store in real-time.
- **Function Calling**: Leverages Gemini's function calling capabilities to allow the AI model to interact with the local file system to create, edit, or delete markdown files in the notes directory.
- **Pluggable Embedding Model**: Uses a local Ollama instance with the `nomic-embed-text` model for generating embeddings, which can be swapped out for other models.

## Architecture

The server is built with a layered architecture:

- **`main.go`**: The entry point of the application. It initializes all dependencies (Gin router, ChromaDB client, Gemini client, services), sets up API routes, and starts the HTTP server.
- **`controller`**: Contains the Gin HTTP handlers that receive requests, validate input, call the appropriate service methods, and return HTTP responses.
- **`services`**: Holds the core business logic of the application.
    - `rag_service.go`: Orchestrates the main RAG pipeline, including embedding text, querying ChromaDB, and generating responses with Gemini.
    - `indexing_service.go`: Manages the lifecycle of file indexing, from initial scanning to real-time watching and updating the vector store.
    - `extractor_service.go`: Handles text extraction from various file formats.
    - `file_actions.go`: Implements the functions for file manipulation that are exposed to the Gemini model.
    - `gemini_tools.go`: Defines the schema for the file action functions available to Gemini.
- **`models`**: Defines the data structures (structs) used for API requests, responses, and internal data representation.

## API Endpoints

All endpoints are prefixed with `/api/v1`.

- **`POST /notes`**: Ingests a new note.
    - **Body**: `{"text": "This is a new note."}`
    - **Response**: `201 Created` with `{"message": "Note ingested successfully"}`
- **`GET /notes`**: Retrieves all ingested notes from the vector store.
    - **Response**: `200 OK` with a JSON object containing the count and a list of notes.
- **`POST /query`**: Queries the RAG pipeline.
    - **Body**: `{"query": "What is the capital of France?"}`
    - **Response**: `200 OK` with a JSON object containing the AI-generated answer and the source documents used for context.
- **`GET /health`**: A health check endpoint.
    - **Response**: `200 OK` with `{"status": "healthy"}`

## Configuration

The server is configured using a `.env` file in the `server` directory. The following environment variables are required:

- `GEMINI_API_KEY`: Your API key for the Google Gemini API.
- `INDEX_PATH`: The absolute or relative path to the directory you want to index and watch for changes (e.g., `../notes`).
- `UNIDOC_LICENSE_KEY`: Your license key for the UniDoc PDF library, required for processing PDF files.

## How to Run

1.  **Install Dependencies**:
    ```bash
    go mod tidy
    ```
2.  **Set up Environment**:
    - Create a `.env` file in the `server` directory.
    - Add the required environment variables (`GEMINI_API_KEY`, `INDEX_PATH`, `UNIDOC_LICENSE_KEY`).
3.  **Run the Server**:
    ```bash
    go run main.go
    ```
The server will start on `http://localhost:8080`.

## Dependencies

- **[Gin](https://github.com/gin-gonic/gin)**: HTTP web framework.
- **[Chroma-Go](https://github.com/amikos-tech/chroma-go)**: Go client for ChromaDB.
- **[Google Gemini Go SDK](https://pkg.go.dev/google.golang.org/genai)**: Go client for the Gemini API.
- **[Ollama](https://ollama.ai/)**: (External) Required for running the local embedding model. Ensure Ollama is running and the `nomic-embed-text` model is pulled (`ollama pull nomic-embed-text`).
- **[UniDoc](https://unidoc.io/)**: Used for extracting text from PDF files.
- **[fsnotify](https://github.com/fsnotify/fsnotify)**: For watching file system events.
