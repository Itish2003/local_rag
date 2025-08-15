import { useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import CircularProgress from '@mui/material/CircularProgress';

function NoteList({ refresh }) {
  const [notes, setNotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/notes')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch notes');
        return res.json();
      })
      .then(data => {
        setNotes(data.notes || []);
        setError('');
      })
      .catch(() => setError('Failed to load notes.'))
      .finally(() => setLoading(false));
  }, [refresh]);

  if (loading) return <CircularProgress sx={{ display: 'block', mx: 'auto', my: 2 }} />;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!notes.length) return <Typography>No notes yet. Add your first note!</Typography>;

  return (
    <Box sx={{ maxHeight: 300, overflowY: 'auto' }}>
      {notes.map((note) => (
        <Card key={note.id} sx={{ mb: 1 }}>
          <CardContent>
            <Typography>{note.text}</Typography>
          </CardContent>
        </Card>
      ))}
    </Box>
  );
}

export default NoteList;
