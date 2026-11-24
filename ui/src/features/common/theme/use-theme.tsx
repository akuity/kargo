import React from 'react';

import { ThemeContext } from './theme-context';

export const useTheme = () => {
  const ctx = React.useContext(ThemeContext);

  if (ctx === null) {
    throw new Error(`useTheme must be used within a ThemeContextProvider`);
  }

  return ctx;
};
