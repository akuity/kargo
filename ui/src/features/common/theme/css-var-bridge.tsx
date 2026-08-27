import { theme } from 'antd';
import React from 'react';

// `colorBgContainer` -> `--kargo-color-bg-container`
const toCssVar = (tokenName: string) =>
  `--kargo-${tokenName.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()}`;

// Republishes AntD's color tokens to :root as CSS vars so non-AntD surfaces
// (sidebar, body, tiles, ...) can read them -- AntD's own cssVar scopes to
// component classes, not :root. Every `color*` token is emitted, so new
// `var(--kargo-color-*)` usages need no edit here. Must live under
// ConfigProvider; useLayoutEffect writes before first paint (no flash).
export const CssVarBridge = () => {
  const { token } = theme.useToken();

  React.useLayoutEffect(() => {
    const root = document.documentElement;
    for (const [name, value] of Object.entries(token)) {
      if (name.startsWith('color') && typeof value === 'string') {
        root.style.setProperty(toCssVar(name), value);
      }
    }
  }, [token]);

  return null;
};
