/**
 * Decode a JWT payload segment per RFC 7519: Base64URL → UTF-8 JSON.
 * Safe for non-ASCII claims and segments using `-`/`_` (unlike atob alone).
 */
export function parseJwtPayload<T = Record<string, unknown>>(token: string): T {
  const parts = token.split('.');
  const segment = parts[1];
  if (!segment) {
    throw new Error('Invalid JWT: missing payload segment');
  }

  const base64 = segment.replace(/-/g, '+').replace(/_/g, '/');
  const pad = (4 - (base64.length % 4)) % 4;
  const padded = base64 + '='.repeat(pad);

  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  const json = new TextDecoder('utf-8').decode(bytes);
  return JSON.parse(json) as T;
}
