# Local RAG API with Go, Gemini, and ChromaDB

This project implements a Retrieval-Augmented Generation (RAG) API using Go, Google's Gemini, ChromaDB for vector storage, and a local Ollama instance for generating embeddings.

## Prerequisites

Before you begin, ensure you have the following installed:
- [Go](https://go.dev/doc/install) (version 1.21 or later)
- [Python](https://www.python.org/downloads/) & [pip](https://pip.pypa.io/en/stable/installation/)
- [Ollama](https://ollama.ai/)

## 1. Setup

### a. Get Your Gemini API Key

1. Go to the [Google AI Studio](https://aistudio.google.com/).
2. Sign in with your Google account.
3. Click on **"Get API key"** and create a new API key in a new or existing project.
4. Copy the generated API key.

### b. Set Environment Variable

You need to set the `GEMINI_API_KEY` environment variable for the Go application to authenticate with the Gemini API. You can do this by exporting it in your shell configuration file (e.g., `.zshrc`, `.bashrc`) or by setting it in the terminal session where you'll run the server.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
```
Replace `"YOUR_GEMINI_API_KEY"` with the key you obtained.

## 2. Running the Services

The application requires three separate services to be running: ChromaDB, Ollama, and the Go backend server.

### a. Run ChromaDB Locally

First, you need to install the `chromadb` Python package.

```bash
pip install chromadb
```

Once installed, open a new terminal window and run the following command to start the ChromaDB server. This command tells Chroma to persist its data to the `./my_chroma_data` directory, which will be created if it doesn't exist.

```bash
chroma run --path ./my_chroma_data
```
Keep this process running.

### b. Run the Ollama Server

In another terminal window, start the Ollama server. This will host the local model needed for generating text embeddings.

```bash
ollama serve
```

Once the server is running, open a **third** terminal window and pull the `nomic-embed-text` model. This is the specific model the Go application is configured to use for embeddings.

```bash
ollama pull nomic-embed-text:v1.5
```
Keep the `ollama serve` process running.

### c. Run the Go Backend Server

Finally, navigate to the `server` directory of the project and start the Go application.

```bash
cd server
go run main.go
```

You should see log messages indicating that the server has started successfully on port 8080 and has connected to the Gemini and ChromaDB services.

## 3. Using the API

The API provides endpoints to ingest notes, ask questions, and retrieve all stored notes. You can use a tool like `curl` or Postman to interact with the API.

Refer to the `endpoints.json` file for a detailed list of endpoints and their corresponding JSON payloads.

### Example `curl` Commands

#### Ingest a Note
```bash
curl -X POST http://localhost:8080/api/v1/notes \
-H "Content-Type: application/json" \
-d '{ 
  "text": "The sky is blue during a clear day."
}'
```


#### Query the RAG Pipeline
```bash
curl -X POST http://localhost:8080/api/v1/query \
-H "Content-Type: application/json" \
-d '{ "query": "What color is the sky?" }'
```


#### Get All Ingested Notes
```bash
curl -X GET http://localhost:8080/api/v1/notes
```

#### Health Check
```bash
curl -X GET http://localhost:8080/health
```

