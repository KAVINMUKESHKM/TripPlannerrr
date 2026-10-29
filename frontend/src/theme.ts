import { createTheme } from "@mui/material/styles";

const theme = createTheme({
  palette: {
    mode: "light",
    primary: { main: "#22333b", dark: "#0a0908", light: "#c6ac8f", contrastText: "#eae0d5" },
    secondary: { main: "#5e503f", dark: "#443a2f", light: "#c6ac8f" },
    background: { default: "#eae0d5", paper: "#fffaf4" },
    text: { primary: "#0a0908", secondary: "#5e503f" },
    divider: "#c6ac8f",
    error: { main: "#b3261e" },
    success: { main: "#5e503f" },
  },
  typography: {
    fontFamily: 'Roboto, "Helvetica Neue", Arial, sans-serif',
    h1: { fontWeight: 600, letterSpacing: "-0.04em" },
    h2: { fontWeight: 600, letterSpacing: "-0.025em" },
    h3: { fontWeight: 600, letterSpacing: "-0.015em" },
    button: { fontWeight: 600, textTransform: "none" },
  },
  shape: { borderRadius: 9 },
  components: {
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: { root: { minHeight: 40, borderRadius: 9, paddingInline: 15, transition: "background-color 180ms ease, border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease" } },
    },
    MuiPaper: { styleOverrides: { root: { backgroundImage: "none" } } },
    MuiCard: { styleOverrides: { root: { borderRadius: 12, boxShadow: "none" } } },
    MuiOutlinedInput: {
      styleOverrides: {
        root: { height: 56, boxSizing: "border-box", borderRadius: 9, backgroundColor: "#fffaf4" },
        notchedOutline: { borderColor: "#c6ac8f" },
      },
    },
    MuiChip: { styleOverrides: { root: { borderRadius: 8, fontWeight: 500 } } },
    MuiAlert: { styleOverrides: { root: { borderRadius: 9 } } },
    MuiTab: { styleOverrides: { root: { minHeight: 42, textTransform: "none", fontWeight: 600 } } },
  },
});

export default theme;
