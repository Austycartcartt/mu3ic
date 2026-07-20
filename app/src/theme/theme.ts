// Single shared style palette. No dark mode / provider yet — plain object,
// per PROJECT.md's "React Native StyleSheet, no styling libraries" rule.
export const theme = {
  colors: {
    background: '#fff',
    surface: '#f5f5f5',
    text: '#111',
    textMuted: '#666',
    border: '#ccc',
    accent: '#222',
    onAccent: '#fff',
    danger: '#900',
    dangerBackground: '#fdd',
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 12,
    lg: 16,
    xl: 24,
  },
  radii: {
    sm: 4,
    md: 8,
  },
  fontSize: {
    sm: 12,
    md: 14,
    lg: 16,
    xl: 18,
  },
} as const;
