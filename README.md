# Local RAG API with Go, Gemini, and ChromaDB

A Retrieval-Augmented Generation (RAG) application implemented with a Go backend, Google Gemini for LLM responses (via API key), ChromaDB for vector storage, and a local Ollama instance for generating embeddings. The project includes a React frontend (optional) and supports real-time file indexing (`.md`, `.txt`, `.pdf`) plus Gemini function-calling to modify local files.

---

## Table of Contents

1.  [Architecture Overview](#architecture-overview)
2.  [Project Structure](#project-structure)
3.  [Prerequisites](#prerequisites)
4.  [Configuration](#configuration)
5.  [Running the Services](#running-the-services)
6.  [API Endpoints & Examples](#api-endpoints--examples)
7.  [Troubleshooting](#troubleshooting)
8.  [Contributing](#contributing)

---

## Architecture Overview

This application is composed of several key services that work together to provide a seamless RAG experience.

```
+-----------------+      +------------------+      +-----------------+
|                 |      |                  |      |                 |
| React Frontend  |<---->|    Go Backend    |<---->|   Google Gemini |
| (Client/Obsidian|      | (Orchestrator)   |      |   (LLM API)     |
|      UI)        |      |                  |      |                 |
+-----------------+      +--------+---------+      +-----------------+
                                  |
                                  |
                  +---------------+----------------+
                  |                                |
        +---------v---------+            +---------v---------+
        |                   |            |                   |
        |      Ollama       |            |     ChromaDB      |
        | (Local Embeddings)|
        |                   |
        +-------------------+            +-------------------+
```

1.  **Frontend (React/Obsidian)**: The user interacts with the system through either the web interface or the Obsidian plugin. It sends user queries to the Go backend.
2.  **Go Backend**: This is the core of the application. It receives requests, orchestrates the RAG pipeline, and manages file indexing.
    - When a **query** is received, it first generates embeddings for the query text using Ollama.
    - It then uses these embeddings to search for relevant documents in ChromaDB.
    - Finally, it sends the query and the retrieved context to Google Gemini to generate a final, context-aware response.
    - When a **document** is added or changed, it uses Ollama to generate embeddings for the content and stores the vectors in ChromaDB.
3.  **Ollama**: A local service that runs the `nomic-embed-text` model to generate vector embeddings for text data. Keeping this local ensures data privacy.
4.  **ChromaDB**: A vector database that stores the embeddings of your documents, enabling efficient similarity searches.
5.  **Google Gemini**: The powerful LLM that provides the generative capabilities, including text generation, function calling, and multi-modal analysis.

---

## Project Structure

The project is organized into the following main directories:

-   `client/`: Contains the source code for the standalone React frontend application.
-   `server/`: Contains the Go backend application, including all API controllers, services, and models.
-   `notes/`: The default directory that the backend monitors for file changes. Your markdown, text, and PDF files go here to be indexed.
-   `Librarian/`: An Obsidian vault containing the plugin that provides a chat interface within Obsidian.
-   `my_chroma_data/`: The default directory where ChromaDB persists its vector data.

---

## Features

- **High-Performance Go Backend**: Engineered for speed and reliability, the Go backend serves as the application\'s central nervous system. It exposes a clean RESTful API, handles concurrent requests with ease, and masterfully orchestrates the complex workflows between the language model, the vector database, and your local file system.
- **Intuitive React Frontend**: A sleek and modern user interface built with React and Material-UI provides a fluid and responsive user experience. The design prioritizes clarity and ease of use, allowing you to focus on your thoughts and ideas without distraction.
- **Seamless Obsidian Integration**: Bridge the gap between your personal knowledge base and powerful AI. The dedicated Obsidian plugin provides a native chat interface directly within your vault. You can converse with your notes, generate new insights, and insert AI-crafted content directly into your editor without ever breaking your creative flow.
- **Private & Local Embeddings with Ollama**: Your data privacy is paramount. The system generates text embeddings entirely on your local machine using Ollama and the `nomic-embed-text` model. This ensures your notes and proprietary information never leave your control, eliminating reliance on third-party services for this critical step.
- **Efficient Vector Storage with ChromaDB**: Leveraging ChromaDB for vector storage, the application ensures efficient and scalable management of document embeddings. This allows for lightning-fast retrieval of relevant information, even as your knowledge base grows.
- **Real-time File Indexing**: The application acts as a living extension of your notes. The backend automatically monitors your specified notes directory for any file changes (`.txt`, `.md`, `.pdf`), ensuring the vector database is always synchronized with your latest thoughts and discoveries.

### Agentic RAG: Beyond Simple Q&A

This isn\'t just a retrieval pipeline; it\'s an intelligent agent. By implementing an **agentic** framework, the Language Model can perform complex reasoning. It deconstructs your queries, forms a multi-step plan, retrieves the most relevant documents, and then synthesizes a comprehensive, context-aware answer. It moves beyond passive generation to become an active partner in your thought process.

![Agentic RAG in action](client/src/assets/Agentic%20RAG.png)

### Tool Calling: Your AI Filesystem Assistant

Unleash the LLM from its digital confines. With Gemini\'s powerful function-calling capabilities, the AI gains the ability to interact directly and intelligently with your local file system. You can ask it to *"create a new note summarizing our last conversation,"* or *"find all notes tagged with #project-alpha and consolidate them into a new report,"* and watch as it executes these commands, turning your conversation into tangible action.

![Tool Calling Demo](client/src/assets/Tool%20Calling.png)

### Multi-Modal Vision: Converse with Your Images

Your knowledge base is more than just text. With the power of Google\'s Gemini, you can now have deep conversations about your images. Drop in a complex diagram, a screenshot of a bug, or a photo of a whiteboard session. Ask the AI to analyze its components, explain a concept, or even transcribe handwritten notes. This multi-modal capability unlocks a new dimension of interaction with your visual information.

![Multi-Modal Image Analysis](client/src/assets/Verbal%20Image%20Analysis.png)

---

## Prerequisites

Make sure these are installed before you start:

*   **Go** (1.21+)
*   **Node.js & npm** (for the React client)
*   **Python & pip** (for ChromaDB)
*   **Ollama** (local model server)
*   A **Google API key** for Gemini (obtainable from Google AI Studio)

## Configuration

### 1. Obtain your Gemini API key

1.  Visit [Google AI Studio](https://aistudio.google.com/) and sign in with your Google account.
2.  Click **"Get API key"** and create an API key under a project.
3.  Copy the generated API key.

Set the environment variable in your shell (or add to `server/.env` as shown below):

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"```

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

> **Note:** If you prefer exporting the env var directly in your shell session (rather than `.env`), that\'s fine too.

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

The app uses **four** main processes that need to be running concurrently: ChromaDB, Ollama, the Go backend, and a frontend (either the React app or Obsidian). Open a separate terminal for each of the first three.

### Terminal 1 — ChromaDB (Vector Store)

This service stores the vector representations of your notes.

1.  **Install ChromaDB**:
    ```bash
    pip install chromadb
    ```
2.  **Run the Server**: This command tells Chroma to persist its data to the `./my_chroma_data` directory.
    ```bash
    chroma run --path ./my_chroma_data
    ```
Leave this terminal running.

### Terminal 2 — Ollama (Local Embeddings)

This service generates the embeddings for your text.

1.  **Start the Ollama Server**:
    ```bash
    ollama serve
    ```
2.  **Pull the Embedding Model**: In a **new, separate terminal**, pull the `nomic-embed-text` model. You only need to do this once.
    ```bash
    ollama pull nomic-embed-text:v1.5
    ```
Keep the `ollama serve` terminal running.

### Terminal 3 — Go Backend

This is the main application server.

1.  **Navigate to the server directory**:
    ```bash
    cd server
    ```
2.  **Run the application**: Make sure your `GEMINI_API_KEY` is set in your environment or in a `.env` file.
    ```bash
    go run main.go
    ```
You should see logs showing the server starting on port `8080` and successfully connecting to the other services.

### Terminal 4 — Frontend (Choose One)

#### Option A: React Web Client

> **Disclaimer:** My primary focus is on developing the Obsidian Plugin as your foremost UI. The client directory was just an experimental UI interface to test the endpoints. It is not being actively updated with the new functionalities. It still exist as a template for anybody to use as a springboard to create a web application rather than going in the direction of plugin.

If you want to use the web interface:

1.  **Navigate to the client directory**:
    ```bash
    cd client
    ```
2.  **Install dependencies and run**:
    ```bash
    npm install
    npm run dev
    ```
The client will be available at `http://localhost:5173`.

#### Option B: Obsidian Plugin

1.  **Open the Vault**: Open the `Librarian/` directory as a vault in your Obsidian application.
2.  **Enable the Plugin**:
    *   Go to `Settings` > `Community Plugins`.
    *   Make sure "Restricted mode" is turned off.
    *   You should see a plugin named "Librarian" under "Installed plugins". Enable it.
3.  **Open the Plugin View**: Click the "brain-circuit" icon in the left ribbon to open the Local RAG chat panel.

---

## API endpoints & examples

The backend exposes several endpoints for interacting with the RAG system.

### Ingest a note

-   **Endpoint**: `POST /api/v1/notes`
-   **Description**: Indexes a new piece of text. The server will generate embeddings, store them in ChromaDB, and save the note to the `INDEX_PATH`.
-   **Example**:
    ```bash
    curl -X POST http://localhost:8080/api/v1/notes \
      -H "Content-Type: application/json" \
      -d '{ "text": "The sky is blue during a clear day." }'
    ```
-   **Success Response** (`201 Created`):
    ```json
    {
      "message": "Note created and indexed successfully"
    }
    ```

### Query the RAG pipeline

-   **Endpoint**: `POST /api/v1/query`
-   **Description**: Sends a query to the RAG pipeline. The backend retrieves relevant context from ChromaDB and uses Gemini to generate a final answer.
-   **Example**:
    ```bash
    curl -X POST http://localhost:8080/api/v1/query \
      -H "Content-Type: application/json" \
      -d '{ "query": "What color is the sky?" }'
    ```
-   **Success Response** (`200 OK`):
    ```json
    {
      "response": "The sky is typically blue on a clear day.",
      "sources": [
        { "id": "doc1", "text": "The sky is blue during a clear day." }
      ]
    }
    ```

### Get all ingested notes

-   **Endpoint**: `GET /api/v1/notes`
-   **Description**: Retrieves a list of all notes currently stored in the system.
-   **Example**:
    ```bash
    curl -X GET http://localhost:8080/api/v1/notes
    ```
-   **Success Response** (`200 OK`):
    ```json
    [
      { "id": "doc1", "text": "The sky is blue during a clear day." },
      { "id": "doc2", "text": "The grass is green." }
    ]
    ```

### Health check

-   **Endpoint**: `GET /health`
-   **Description**: A simple health check endpoint to verify the backend is running.
-   **Example**:
    ```bash
    curl -X GET http://localhost:8080/health
    ```
-   **Success Response** (`200 OK`):
    ```json
    {
      "status": "ok"
    }
    ```

---

## Troubleshooting

*   **Gemini API Key Issues**: If you get authentication errors, ensure your `GEMINI_API_KEY` is correctly set in `server/.env` or as an environment variable. Double-check for typos.
*   **Ollama Connection Errors**: If the backend can\'t reach Ollama, make sure the `ollama serve` process is running in its own terminal. Verify that the `nomic-embed-text` model has been pulled successfully.
*   **ChromaDB Errors**: If you see errors related to ChromaDB, ensure the `chroma run --path ./my_chroma_data` command is running. Check for file permission issues in the `my_chroma_data` directory.
*   **PDF Indexing Failures**: For robust PDF parsing, a UniDoc license key may be required. If you are working with complex PDFs and they are not being indexed correctly, consider obtaining a license and setting the `UNIDOC_LICENSE_KEY` in your environment.
*   **Check the Logs**: The output from `go run main.go` is the best place to look for errors. The logs will usually indicate which part of the process is failing (e.g., embedding generation, ChromaDB upsert, or the Gemini API call).

---

## Contributing

Contributions are welcome! Whether it\'s reporting a bug, suggesting a new feature, or submitting a pull request, your input is valued.

1.  **Reporting Bugs**: Please open an issue on the GitHub repository, providing as much detail as possible, including steps to reproduce the bug.
2.  **Feature Requests**: Open an issue to discuss your idea. This allows for feedback and ensures it aligns with the project\'s goals.
3.  **Pull Requests**:
    *   Fork the repository.
    *   Create a new branch for your feature or bug fix.
    *   Make your changes and commit them with clear, descriptive messages.
    *   Push your branch and open a pull request.

Please ensure your code adheres to the existing style and conventions.