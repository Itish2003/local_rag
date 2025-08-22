# Local RAG API with Go, Gemini, and ChromaDB

A Retrieval-Augmented Generation (RAG) application implemented with a Go backend, Google Gemini for LLM responses (via API key), ChromaDB for vector storage, and a local Ollama instance for generating embeddings. The project includes a React frontend (optional) and supports real-time file indexing (`.md`, `.txt`, `.pdf`) plus Gemini function-calling to modify local files.

---

## Features

- **Go Backend**: A robust backend powered by Go, providing a RESTful API for ingesting notes, querying the RAG pipeline, and managing documents.
- **React Frontend**: An intuitive user interface built with React and Material-UI, allowing for easy interaction with the backend services.
- **Retrieval-Augmented Generation (RAG)**: Leverages the power of RAG to provide context-aware answers by retrieving relevant documents from a vector store before generating a response.
- **Local Embeddings with Ollama**: Generates text embeddings locally using Ollama and the `nomic-embed-text` model, ensuring data privacy and reducing reliance on external services.
- **Vector Storage with ChromaDB**: Uses ChromaDB for efficient storage and retrieval of document embeddings.
- **Google Gemini Integration**: Integrates with the Google Gemini API for powerful language model capabilities, including text generation and function calling.
- **Real-time File Indexing**: The backend automatically monitors a specified directory for file changes (`.txt`, `.md`, `.pdf`), keeping the vector database continuously synchronized.
- **Gemini Function Calling**: The application empowers the LLM to interact directly with the local file system, allowing it to create, edit, and delete markdown files based on user prompts.

---

## Table of contents

1. [Prerequisites](#prerequisites)
2. [Configuration](#configuration)
3. [Running the services](#running-the-services)
4. [API endpoints & examples](#api-endpoints--examples)
5. [Notes, tips & troubleshooting](#notes-tips--troubleshooting)

---

## Prerequisites

Make sure these are installed before you start:

* **Go** (1.21+)
* **Node.js & npm** (for the React client)
* **Python & pip** (for ChromaDB)
* **Ollama** (local model server)
* A **Google API key** for Gemini (obtainable from Google AI Studio)

## Configuration

### 1. Obtain your Gemini API key

1. Visit [Google AI Studio](https://aistudio.google.com/) and sign in with your Google account.
2. Click **"Get API key"** and create an API key under a project.
3. Copy the generated API key.

Set the environment variable in your shell (or add to `server/.env` as shown below):

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
```

### 2. Backend environment file (`server/.env`)

Create a `.env` file inside the `server/` directory with the following variables:

```env
# Get from Google AI Studio
GEMINI_API_KEY="YOUR_GEMINI_API_KEY"

# Path to your notes directory (used by the file indexer)
INDEX_PATH="../notes"

# (Optional) UniDoc Cloud key if your pipeline uses UniDoc for PDFs
UNIDOC_LICENSE_KEY="YOUR_UNIDOC_LICENSE_KEY"
```

> **Note:** If you prefer exporting the env var directly in your shell session (rather than `.env`), that's fine too.

### 3. Frontend proxy (`client/vite.config.js`)

If you run the optional React frontend, create a Vite config to proxy API calls to the Go backend:

```javascript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

---

## Running the services

The app uses **four** processes: ChromaDB, Ollama, the Go backend, and (optionally) the React frontend. Open separate terminals for each.

### Terminal 1 — ChromaDB (vector store)

Install the Python package and start the Chroma server (persistent path recommended):

```bash
pip install chromadb
```

Once installed, open a new terminal window and run the following command to start the ChromaDB server. This command tells Chroma to persist its data to the `./my_chroma_data` directory, which will be created if it doesn't exist.

```bash
chroma run --path ./my_chroma_data
```

Leave this running. The `--path` directory will be created if it doesn't exist.

### Terminal 2 — Ollama (local embeddings)

In another terminal window, start the Ollama server. This will host the local model needed for generating text embeddings.

```bash
ollama serve
```

Once the server is running, open a **third** terminal window and pull the `nomic-embed-text` model. This is the specific model the Go application is configured to use for embeddings.

```bash
ollama pull nomic-embed-text:v1.5
```

Keep `ollama serve` running.

### Terminal 3 — Go backend

Finally, navigate to the `server` directory of the project and start the Go application.

```bash
cd server
# ensure GEMINI_API_KEY is set (or present in server/.env)
go run main.go
```

You should see logs showing the server starting on port `8080` and successful connections to Gemini/Chroma/Ollama.

### Terminal 4 — React frontend (optional)

If you want the web client:

```bash
cd client
npm install
npm run dev
```

By default Vite serves the client at `http://localhost:5173` and proxies `/api` to the Go backend.

---

## API endpoints & examples

The backend exposes endpoints for ingesting notes, running a RAG-style query, retrieving stored notes, and health checks. Refer to `endpoints.json` in the repo for the authoritative list and any additional payload examples.

Below are the most commonly used `curl` examples:

### Ingest a note

```bash
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{ "text": "The sky is blue during a clear day." }'
```

**Payload**

* `text` — the string content to index (plain text). The server will generate embeddings, store vectors in ChromaDB, and (if configured) save the note to the `INDEX_PATH`.

### Query the RAG pipeline

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{ "query": "What color is the sky?" }'
```

**Payload**

* `query` — the user question. The backend will retrieve relevant chunks from ChromaDB, and call Gemini to generate the final answer.

### Get all ingested notes

```bash
curl -X GET http://localhost:8080/api/v1/notes
```

Returns a JSON array of stored notes (metadata and/or raw text depending on server implementation).

### Health check

```bash
curl -X GET http://localhost:8080/health
```

A simple `200 OK` JSON response indicates the backend is running.

---

## Notes, tips & troubleshooting

* If the backend can't reach Gemini, confirm `GEMINI_API_KEY` is set properly and your machine has network access.
* If embeddings fail, ensure `ollama serve` is running and the `nomic-embed-text` model is pulled.
* If Chroma can't persist data, check file permissions in the directory passed to `chroma run --path`.
* Logs from `go run main.go` will usually indicate where the flow fails (embedding generation, Chroma upsert, or Gemini call).
* For PDF parsing, you may need a UniDoc license if the project uses UniDoc for robust PDF extraction. Configure `UNIDOC_LICENSE_KEY` as needed.

---

## Want changes?

If you'd like the README adjusted (more examples, additional endpoints added from `endpoints.json`, or a CONTRIBUTING section), tell me what to include and I’ll update the document.
