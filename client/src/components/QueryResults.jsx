import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';

function QueryResults({ result, loading, error }) {
  if (loading) return <Typography>Loading...</Typography>;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!result) return null;

  return (
    <Card sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 1 }}>Answer</Typography>
        <Typography sx={{ mb: 2 }}>{result.answer || result.response || 'No answer.'}</Typography>
        {result.sources && result.sources.length > 0 && (
          <Box>
            <Typography variant="subtitle2">Source Documents:</Typography>
            <ul>
              {result.sources.map((src, idx) => (
                <li key={idx}>
                  <Typography variant="body2">{src}</Typography>
                </li>
              ))}
            </ul>
          </Box>
        )}
      </CardContent>
    </Card>
  );
}

export default QueryResults;
