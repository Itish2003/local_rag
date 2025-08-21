import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Accordion from '@mui/material/Accordion';
import AccordionSummary from '@mui/material/AccordionSummary';
import AccordionDetails from '@mui/material/AccordionDetails';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

function QueryResults({ result, loading, error }) {
  if (loading) return <Typography>Loading...</Typography>;
  if (error) return <Typography color="error">{error}</Typography>;
  if (!result) return null;

  return (
    <Card sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 1 }}>Answer</Typography>
        <Typography sx={{ mb: 3, whiteSpace: 'pre-wrap' }}>{result.answer || result.response || 'No answer.'}</Typography>

        {result.source_docs && result.source_docs.length > 0 && (
          <Box>
            <Typography variant="subtitle1" sx={{ mb: 1 }}>Sources</Typography>
            {result.source_docs.map((doc, idx) => (
              <Accordion key={idx} sx={{ '&:before': { display: 'none' }, boxShadow: 'none', border: '1px solid rgba(0, 0, 0, 0.12)' }}>
                <AccordionSummary
                  expandIcon={<ExpandMoreIcon />}
                  aria-controls={`panel${idx}-content`}
                  id={`panel${idx}-header`}
                >
                  <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                    {/* Display filename from metadata */}
                    Source {idx + 1}: {doc.metadata?.source_file?.split('/').pop() || 'User Note'}
                  </Typography>
                </AccordionSummary>
                <AccordionDetails sx={{ backgroundColor: 'rgba(0, 0, 0, 0.03)' }}>
                  <Typography variant="caption" display="block" color="text.secondary">
                    Chunk {doc.metadata?.chunk_num !== undefined ? doc.metadata.chunk_num + 1 : 'N/A'}
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 1 }}>
                    {doc.text}
                  </Typography>
                </AccordionDetails>
              </Accordion>
            ))}
          </Box>
        )}
      </CardContent>
    </Card>
  );
}

export default QueryResults;

