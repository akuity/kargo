import { createContext } from 'react';

import { ResolvedTheme, ThemePreference } from '@ui/features/theme/types';

type ThemeContextValue = {
  preference: ThemePreference;
  resolvedTheme: ResolvedTheme;
  setPreference: (preference: ThemePreference) => void;
};

export const ThemeContext = createContext<ThemeContextValue | null>(null);
