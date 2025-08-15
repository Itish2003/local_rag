import Header from './components/Header';
import Footer from './components/Footer';
import NoteInputForm from './components/NoteInputForm';
import NoteList from './components/NoteList';
import QueryInput from './components/QueryInput';
import QueryResults from './components/QueryResults';
import Box from '@mui/material/Box';
import Container from '@mui/material/Container';
import Paper from '@mui/material/Paper';
import { useState } from 'react';

function App() {
  const [refreshNotes, setRefreshNotes] = useState(0);
  const [queryResult, setQueryResult] = useState(null);
  const [queryLoading, setQueryLoading] = useState(false);
  const [queryError, setQueryError] = useState('');

  const handleNoteAdded = () => setRefreshNotes(r => r + 1);

  const handleQuery = async (query) => {
    setQueryLoading(true);
    setQueryError('');
    setQueryResult(null);
    try {
      const res = await fetch('/api/v1/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query }),
      });
      if (!res.ok) throw new Error('Query failed');
      const data = await res.json();
      setQueryResult(data);
    } catch (err) {
      setQueryError('Failed to get results.');
    } finally {
      setQueryLoading(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header />
      <Container maxWidth="lg" sx={{ flex: 1, py: 4 }}>
        <Paper sx={{ display: 'flex', gap: 4, p: 2, minHeight: 500 }}>
          {/* Left Column: Note Ingestion */}
          <Box sx={{ flex: 1, pr: 2, borderRight: (theme) => `1px solid ${theme.palette.divider}` }}>
            <NoteInputForm onNoteAdded={handleNoteAdded} />
            <NoteList refresh={refreshNotes} />
          </Box>
          {/* Right Column: Query and Results */}
          <Box sx={{ flex: 2, pl: 2 }}>
            <QueryInput onQuery={handleQuery} />
            <QueryResults result={queryResult} loading={queryLoading} error={queryError} />
          </Box>
        </Paper>
      </Container>
      <Footer />
    </Box>
  );
}

export default App;
