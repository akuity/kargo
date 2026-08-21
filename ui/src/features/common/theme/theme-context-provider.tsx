import { ConfigProvider } from 'antd';
import React, { PropsWithChildren } from 'react';

import { themeConfig } from '@ui/config/themeConfig';

import { CssVarBridge } from './css-var-bridge';
import { ThemeContext, ThemeContextType, ThemeMode } from './theme-context';

const themeModeKey = 'theme-mode';

const prefersDarkQuery = '(prefers-color-scheme: dark)';

const systemPrefersDark = (): boolean =>
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia(prefersDarkQuery).matches;

const readStoredMode = (): ThemeMode => {
  const stored = localStorage.getItem(themeModeKey);
  if (stored === 'light' || stored === 'dark' || stored === 'system') {
    return stored;
  }
  return 'system';
};

export const ThemeContextProvider = ({ children }: PropsWithChildren) => {
  const [mode, setModeState] = React.useState<ThemeMode>(readStoredMode);
  const [systemDark, setSystemDark] = React.useState(systemPrefersDark);

  // Track the OS preference so `system` mode stays in sync while the app is open.
  React.useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const mediaQuery = window.matchMedia(prefersDarkQuery);
    const handleChange = (event: MediaQueryListEvent) => setSystemDark(event.matches);
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const isDark = mode === 'system' ? systemDark : mode === 'dark';

  const setMode = React.useCallback((next: ThemeMode) => {
    localStorage.setItem(themeModeKey, next);
    setModeState(next);
  }, []);

  // `data-theme` on <html> is the single signal Tailwind, our CSS vars, and
  // native controls (color-scheme) all read. Mirror the pre-paint index.html.
  React.useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', isDark ? 'dark' : 'light');
    root.style.colorScheme = isDark ? 'dark' : 'light';
  }, [isDark]);

  const ctx: ThemeContextType = React.useMemo(
    () => ({ mode, isDark, setMode }),
    [mode, isDark, setMode]
  );

  // Owns the AntD theme too, so `isDark` comes from local state, not context.
  return (
    <ThemeContext.Provider value={ctx}>
      <ConfigProvider theme={themeConfig(isDark)}>
        <CssVarBridge />
        {children}
      </ConfigProvider>
    </ThemeContext.Provider>
  );
};
