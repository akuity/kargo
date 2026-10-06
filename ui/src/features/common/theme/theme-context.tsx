import React from 'react';

// User preference; `system` follows prefers-color-scheme.
export type ThemeMode = 'light' | 'dark' | 'system';

export interface ThemeContextType {
  mode: ThemeMode; // selected preference (may be `system`)
  isDark: boolean; // resolved appearance
  setMode: (mode: ThemeMode) => void;
}

export const ThemeContext = React.createContext<ThemeContextType | null>(null);
