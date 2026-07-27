/**
 * Runtime helpers for the kargo UI's URL path prefix (basePath).
 *
 * The kargo API server may be deployed at a URL path prefix (e.g. /kargo).
 * When it is, the served index.html carries a window.__KARGO_BASE_PATH__
 * global populated by the server at serve time. The functions here read
 * that global and use it to compose URLs the UI hands to the browser
 * (REST baseUrl, fetch URLs, React Router basename, and the
 * window.location.replace() destinations used by auth flows).
 *
 * When the server is deployed at the root (no basePath), the global is
 * absent or empty and these helpers degrade to identity.
 */

declare global {
  interface Window {
    __KARGO_BASE_PATH__?: string;
  }
}

/**
 * basePath returns the URL path prefix the kargo API server is deployed
 * at, with no trailing slash. Empty string when the server lives at the
 * root.
 */
export const basePath = (): string => window.__KARGO_BASE_PATH__ ?? '';

/**
 * withBasePath composes a root-relative URL path with the runtime basePath.
 * Use at every call site that builds an absolute path the browser will
 * navigate to or fetch from. Idempotent on the empty-basePath case.
 */
export const withBasePath = (path: string): string => {
  const base = basePath();
  if (!base) return path;
  if (path.startsWith('/')) return base + path;
  return `${base}/${path}`;
};

export {};
