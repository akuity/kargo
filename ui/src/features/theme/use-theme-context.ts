import { useContext } from 'react';

import { ThemeContext } from '@ui/features/theme/theme-context';

export const useThemeContext = () => {
  const context = useContext(ThemeContext);

  if (!context) {
    throw new Error('missing ThemeContext');
  }

  return context;
};
