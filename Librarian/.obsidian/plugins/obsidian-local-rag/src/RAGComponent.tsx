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
    useEffect(() => {
        if (iconRef.current) {
            setIcon(iconRef.current, "plus-square"); // Use a built-in Obsidian icon
        }
    }, []);
    return <div ref={iconRef} className="insert-icon" onClick={onClick} title="Insert into note"></div>;
};

const ClearButton = ({ onClick }: { onClick: () => void }) => {
    const iconRef = useRef<HTMLDivElement>(null);
    useEffect(() => {
        if (iconRef.current) setIcon(iconRef.current, "trash-2"); // Use a trash icon
    }, []);
    return <div ref={iconRef} className="clear-icon" onClick={onClick} title="Clear chat history and start a new conversation"></div>;
};

export const RAGComponent = ({ settings, events }: RAGComponentProps) => {
  // --- STATE and REFs ---
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [currentQuery, setCurrentQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null); 
  const chatHistoryRef = useRef<HTMLDivElement>(null);
  const queryInputRef = useRef<HTMLTextAreaElement>(null); // Ref for the textarea
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
  }, [events, settings,sessionID]); // Re-run this effect if the events or settings objects change


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
    queryInputRef.current?.focus();
  };

  const handleSubmitQuery = async (queryText?: string) => {
    // Use the explicitly passed queryText or fall back to the one in the state
    const query = (queryText || currentQuery).trim();
    if (!query || loading) return;
    events.emit('query-start');


    // Add the user's message to the chat history immediately for a responsive feel
    const userMessage: ChatMessage = { role: 'user', content: query };
    setMessages(prev => [...prev, userMessage]);
    
    // Clear the input field and set loading state
    setCurrentQuery('');
    setLoading(true);

    try {
      // Call the backend using the URL from settings
      const response = await requestUrl({
        url: `${settings.backendUrl}/api/v1/query`,
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query,sessionID }),
      });

      const assistantResponse = response.json;
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
      // Refocus the input area after a query for a smoother workflow
      queryInputRef.current?.focus();
      events.emit('query-end'); // Notify that the query has ended
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
                {indexStatus ? 
                  `Indexed: ${indexStatus.totalFiles} files (${indexStatus.totalChunks} chunks)` : 
                  'Index status: unavailable'
                }
            </div>
            <div className="header-actions"> {/* <-- Moved back inside */}
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
                  <details key={idx}>
                    <summary>
                      {doc.metadata?.source_file?.split('/').pop() || 'User Note'}
                    </summary>
                    <div className="rag-sources-content">
                      <p>{doc.text}</p>
                    </div>
                  </details>
                ))}
              </div>)}
              <div className="message-actions">
                    <InsertButton onClick={() => handleInsert(msg.content)} />
                </div>
            </>
            )}
          </div>
        ))}
        {loading && (
             <div className="chat-message assistant-message">Thinking...</div>
        )}
      </div>
      <div className="chat-input-form">
        <textarea
          ref={queryInputRef} // Attach ref to the textarea element
          placeholder="Ask a question about your notes..."
          value={currentQuery}
          onChange={(e) => setCurrentQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={loading}
          rows={2}
        />
        <button onClick={() => handleSubmitQuery()} disabled={loading}>
          Send
        </button>
      </div>
    </div>
  );
};