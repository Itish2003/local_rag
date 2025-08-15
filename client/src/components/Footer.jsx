import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

function Footer() {
  return (
    <Box component="footer" sx={{ mt: 4, py: 2, textAlign: 'center', bgcolor: 'background.paper' }}>
      <Typography variant="body2" color="text.secondary">
        <a href="https://github.com/your-repo" target="_blank" rel="noopener noreferrer">
          View on GitHub
        </a>
      </Typography>
    </Box>
  );
}

export default Footer;
