import { useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

function NoteInputForm({ onNoteAdded }) {
  const [note, setNote] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!note.trim()) {
      setError('Note cannot be empty.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const res = await fetch('/api/v1/notes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: note }),
      });
      if (!res.ok) throw new Error('Failed to add note');
      setNote('');
      onNoteAdded();
    } catch (err) {
      setError('Failed to add note.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField
            label="Add a new note"
            multiline
            minRows={2}
            fullWidth
            value={note}
            onChange={e => setNote(e.target.value)}
            error={!!error}
            helperText={error}
            disabled={loading}
            sx={{ mb: 2 }}
          />
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? <CircularProgress size={24} /> : 'Add Note'}
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
}

export default NoteInputForm;
