import { requestUrl, setIcon } from 'obsidian';
import * as React from 'react';
import { useState, useRef, useEffect } from 'react';
import { LocalRAGSettings } from '../main';
import { EventEmitter } from './EventEmitter';
import ReactMarkdown from 'react-markdown'; // <-- NEW: Import Markdown renderer
import remarkGfm from 'remark-gfm'; 

// --- Interface Definitions ---

// The structure of a single message in our chat history
interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  sources?: SourceDoc[];
}

// The structure of a source document from the backend
interface SourceDoc {
  text: string;
  metadata?: {
    source_file?: string;
    chunk_num?: number;
  };
}

// The shape of the props our component is receiving from RAGView.tsx
interface RAGComponentProps {
    settings: LocalRAGSettings;
    events: EventEmitter;
}

interface IndexStatus {
    totalFiles: number;
    totalChunks: number;
}

// --- Helper Component for the Insert Icon ---
const InsertButton = ({ onClick }: { onClick: () => void }) => {
    const iconRef = useRef<HTMLDivElement>(null);
    useEffect(() => { if (iconRef.current) setIcon(iconRef.current, "plus-square"); }, []);
    return <div ref={iconRef} className="insert-icon" onClick={onClick} title="Insert into note"></div>;
};

const ClearButton = ({ onClick }: { onClick: () => void }) => {
    const iconRef = useRef<HTMLDivElement>(null);
    useEffect(() => { if (iconRef.current) setIcon(iconRef.current, "trash-2"); }, []);
    return <div ref={iconRef} className="clear-icon" onClick={onClick} title="Clear chat history and start a new conversation"></div>;
};

const AttachmentButton = ({ onClick }: { onClick: () => void }) => {
    const iconRef = useRef<HTMLDivElement>(null);
    useEffect(() => { if (iconRef.current) setIcon(iconRef.current, "paperclip"); }, []);
    return <div ref={iconRef} className="attachment-icon" onClick={onClick} title="Attach a file"></div>;
};

export const RAGComponent = ({ settings, events }: RAGComponentProps) => {
  // --- STATE and REFs ---
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [currentQuery, setCurrentQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null); 
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const chatHistoryRef = useRef<HTMLDivElement>(null);
  const queryInputRef = useRef<HTMLTextAreaElement>(null); 
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [sessionID, setSessionID] = useState<string>('');

  // --- EFFECTS ---

  // Effect to auto-scroll the chat history to the bottom when new messages are added
  useEffect(() => {
    chatHistoryRef.current?.scrollTo(0, chatHistoryRef.current.scrollHeight);
  }, [messages]);

  // Effect to listen for external 'new-query' events from the editor commands
  useEffect(() => {
    const handleNewQuery = (text: string) => {
      // Set the query in the input box and immediately submit it
      setCurrentQuery(text); 
      handleSubmitQuery(text); // Pass text directly to avoid React state update lag
    };

    events.on('new-query', handleNewQuery);

    // Cleanup function: remove the listener when the component is unmounted
    return () => {
      events.off('new-query', handleNewQuery);
    };
  }, [events, settings,sessionID,selectedFile]); // Re-run this effect if the events or settings objects change


   useEffect(() => {
    const fetchIndexStatus = async () => {
        try {
            // NOTE: You need to create this GET /api/v1/status endpoint on your Go backend
            const response = await requestUrl({ url: `${settings.backendUrl}/api/v1/status` });
            setIndexStatus(response.json);
        } catch (error) {
            console.warn("Could not fetch index status. Is the backend running a version with the /api/v1/status endpoint?");
            setIndexStatus(null);
        }
    };
    fetchIndexStatus(); // Fetch on component mount
    const intervalId = setInterval(fetchIndexStatus, 30000); // And then every 30 seconds
    return () => clearInterval(intervalId); // Cleanup on unmount
  }, [settings.backendUrl]);

  // --- API CALL FUNCTION ---

  const handleInsert = (textToInsert: string) => {
    events.emit('insert-text', textToInsert);
  };

   const handleClearChat = () => {
    setMessages([]);
    setSessionID(''); // Resetting the session ID starts a new conversation
    setSelectedFile(null);
    queryInputRef.current?.focus();
  };

   const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files.length > 0) {
        setSelectedFile(event.target.files[0]);
    }
  };

  const handleSubmitQuery = async (queryText?: string) => {
    const query = (queryText || currentQuery).trim();
    if (!query || loading) return;
    events.emit('query-start');

    const userMessageContent = selectedFile ? `[File: ${selectedFile.name}] ${query}` : query;
    const userMessage: ChatMessage = { role: 'user', content: userMessageContent };
    setMessages(prev => [...prev, userMessage]);
    
    setCurrentQuery('');
    setLoading(true);

    try {
      const formData = new FormData();
      formData.append('query', query);
      formData.append('sessionID', sessionID);
      if (selectedFile) {
        formData.append('file', selectedFile);
      }
      
      // Use the standard 'fetch' API, which correctly handles FormData.
      const response = await fetch(`${settings.backendUrl}/api/v1/query`, {
        method: 'POST',
        body: formData, // 'fetch' automatically sets the correct Content-Type with boundary.
      });

      if (!response.ok) {
        // 'fetch' does not throw on HTTP errors, so we need to check the status.
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // We need to await the .json() method to parse the response body.
      const assistantResponse = await response.json();

      const assistantMessage: ChatMessage = {
        role: 'assistant',
        content: assistantResponse.answer,
        sources: assistantResponse.source_docs,
      };
      setMessages(prev => [...prev, assistantMessage]);
      setSessionID(assistantResponse.sessionID);

    } catch (err) {
      console.error("RAG query failed:", err);
      const errorMessage: ChatMessage = {
        role: 'assistant',
        content: 'Error: Could not get a response. Please check the Backend Server URL in settings and ensure the server is running.',
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setLoading(false);
      setSelectedFile(null);
      queryInputRef.current?.focus();
      events.emit('query-end');
    }
  };
  
  // Handle 'Enter' key press for quick submission
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault(); // Prevent new line on Enter
        handleSubmitQuery();
    }
  }

  // --- RENDER LOGIC ---
  return (
    <div className="local-rag-container">
        <div className="chat-header">
            <div className="index-status">
                {indexStatus ? `Indexed: ${indexStatus.totalFiles} files (${indexStatus.totalChunks} chunks)` : 'Index status: unavailable'}
            </div>
            <div className="header-actions">
                <ClearButton onClick={handleClearChat} />
            </div>
      </div>
      <div className="chat-history" ref={chatHistoryRef}>
        {messages.map((msg, index) => (
          <div key={index} className={`chat-message ${msg.role}-message`}>
            <div className="assistant-answer"><ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown></div>
            {msg.role === 'assistant' && (
                <>
                {msg.sources && msg.sources.length > 0 && (
              <div className="rag-sources">
                <strong>Sources:</strong>
                {msg.sources.map((doc, idx) => (
                  <details key={idx}><summary>{doc.metadata?.source_file?.split('/').pop() || 'User Note'}</summary><div className="rag-sources-content"><p>{doc.text}</p></div></details>
                ))}
              </div>)}
              <div className="message-actions">
                    <InsertButton onClick={() => handleInsert(msg.content)} />
                </div>
            </>
            )}
          </div>
        ))}
        {loading && (<div className="chat-message assistant-message">Thinking...</div>)}
      </div>
      
      {/* NEW: Display for the currently selected file */}
      {selectedFile && (
        <div className="attached-file-display">
            <span>{selectedFile.name}</span>
            <button onClick={() => setSelectedFile(null)}>✖</button>
        </div>
      )}

      <div className="chat-input-form">
        {/* NEW: Hidden file input and an attachment button to trigger it */}
        <input type="file" ref={fileInputRef} onChange={handleFileChange} style={{ display: 'none' }} />
        <AttachmentButton onClick={() => fileInputRef.current?.click()} />
        <textarea
          ref={queryInputRef}
          placeholder="Ask a question or attach a file..."
          value={currentQuery}
          onChange={(e) => setCurrentQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={loading}
          rows={2}
        />
        <button onClick={() => handleSubmitQuery()} disabled={loading}>Send</button>
      </div>
    </div>
  );
};