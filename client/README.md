# Local RAG Client

This directory contains the React frontend for the Local RAG (Retrieval-Augmented Generation) application. It provides a user interface to interact with the Go backend server, allowing users to ingest notes, view existing notes, and ask questions to the RAG pipeline.

## Features

- **React + Vite**: A modern, fast, and efficient frontend setup.
- **Material-UI**: A popular React UI framework for building beautiful and responsive user interfaces.
- **Two-Column Layout**:
    - **Left Column**: For ingesting new notes and viewing a list of all notes currently in the vector store.
    - **Right Column**: For querying the RAG pipeline and viewing the AI-generated answer along with its sources.
- **API Proxying**: The Vite development server is configured to proxy API requests to the backend Go server, avoiding CORS issues during development.

## Architecture

- **`main.jsx`**: The entry point of the React application. It sets up the Material-UI theme and renders the main `App` component.
- **`App.jsx`**: The root component that manages the overall application state and layout. It contains the logic for handling note creation and querying the backend.
- **`components/`**: This directory contains reusable React components:
    - `Header.jsx`: The main application header.
    - `Footer.jsx`: The application footer.
    - `NoteInputForm.jsx`: A form for submitting new notes to the backend.
    - `NoteList.jsx`: A component that fetches and displays all notes from the backend.
    - `QueryInput.jsx`: A form for submitting queries to the RAG pipeline.
    - `QueryResults.jsx`: A component to display the answer and source documents from a RAG query.

## How to Run

1.  **Install Dependencies**:
    ```bash
    npm install
    ```
2.  **Run the Development Server**:
    ```bash
    npm run dev
    ```
The client application will be available at `http://localhost:5173` (or another port if 5173 is in use).

**Note**: The backend server must be running for the client application to function correctly, as it relies on the backend for all its data.

## Available Scripts

- `npm run dev`: Starts the Vite development server.
- `npm run build`: Builds the application for production.
- `npm run lint`: Lints the codebase using ESLint.
- `npm run preview`: Serves the production build locally for previewing.

## Dependencies

- **React**: A JavaScript library for building user interfaces.
- **Vite**: A modern frontend build tool.
- **Material-UI**: A React UI framework.
- **Axios**: A promise-based HTTP client for making requests to the backend.
- **ESLint**: A tool for identifying and reporting on patterns found in ECMAScript/JavaScript code.