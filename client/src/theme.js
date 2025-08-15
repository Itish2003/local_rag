import { createTheme } from '@mui/material/styles';

// A clean, professional, and minimalist light theme.
const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1976d2', // A classic, professional blue
    },
    secondary: {
      main: '#dc004e', // A contrasting pink for secondary actions if needed
    },
    background: {
      default: '#f4f6f8', // A very light, soft gray
      paper: '#ffffff',   // Pure white for cards and surfaces
    },
    text: {
      primary: '#212121',   // Crisp, dark gray for high readability
      secondary: '#757575', // Lighter gray for secondary text
    },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    h6: {
      fontWeight: 600,
    },
  },
  components: {
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: '#ffffff',
          color: '#212121', // Dark text on a light app bar
          boxShadow: '0 1px 4px rgba(0,0,0,0.1)', // A subtle shadow for depth
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: '8px',
          border: '1px solid #e0e0e0', // A light border for definition
          boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
          borderRadius: '6px',
        },
      },
    },
  },
});

export default theme;
