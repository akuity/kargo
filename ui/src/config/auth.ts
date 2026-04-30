export const authTokenKey = 'auth_token';
export const refreshTokenKey = 'refresh_token';

export const redirectToQueryParam = 'redirectTo';

// Validate that a redirect path is a safe, same-origin relative path.
export const isSafeRedirectPath = (path: string | null): path is string => {
  if (!path || !path.startsWith('/')) return false;
  try {
    return new URL(path, window.location.origin).origin === window.location.origin;
  } catch {
    return false;
  }
};
