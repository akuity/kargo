import type { CaptureResult, Properties } from 'posthog-js';
import { matchRoutes } from 'react-router-dom';

import { basePath } from '@ui/config/base-path';
import { paths } from '@ui/config/paths';

// Every route the UI knows, plus a wildcard beneath each. The wildcards give
// any path that no route names exactly (a settings sub-page, an extension's
// own route, a mistyped URL) the nearest known pattern, down to "/*", so that
// no path is ever reported verbatim.
const routes = [...new Set(Object.values(paths))].flatMap((path) => [
  { path },
  { path: `${path.replace(/\/$/, '')}/*` }
]);

/**
 * routePattern returns the pattern of the route a path belongs to, e.g.
 * "/project/:name" for "/project/my-project". Paths carry the names of
 * Projects, Stages, and other resources; patterns do not.
 */
export const routePattern = (pathname: string): string =>
  matchRoutes(routes, { pathname }, basePath() || '/')?.[0]?.route.path ?? '/*';

const isURL = (value: string): boolean => /^https?:\/\//i.test(value);

// sanitizeURL reduces one of the UI's own URLs to its origin and route
// pattern, and any other URL to its origin alone. Query strings and fragments
// are always dropped: the UI's can carry OIDC authorization codes.
const sanitizeURL = (value: string): string => {
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    return '';
  }
  if (url.origin !== window.location.origin) {
    return url.origin;
  }
  return url.origin + basePath() + routePattern(url.pathname);
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

// The kinds of entry in a session recording that state the page's URL: one
// written when recording starts (4), and PostHog's own notes of each later
// page view or URL change (5).
const recordingMeta = 4;
const recordingCustom = 5;

// sanitizeRecordingEntry sanitizes the page URL in an entry of a session
// recording, so that recordings are listed and filtered by the same route
// patterns as other events. It does not touch what the recording shows of
// the page itself.
const sanitizeRecordingEntry = (entry: unknown): unknown => {
  if (!isRecord(entry) || !isRecord(entry.data)) {
    return entry;
  }
  const { data } = entry;
  if (entry.type === recordingMeta && typeof data.href === 'string') {
    return { ...entry, data: { ...data, href: sanitizeURL(data.href) } };
  }
  if (
    entry.type === recordingCustom &&
    isRecord(data.payload) &&
    typeof data.payload.href === 'string'
  ) {
    const payload = { ...data.payload, href: sanitizeURL(data.payload.href) };
    return { ...entry, data: { ...data, payload } };
  }
  return entry;
};

const sanitizeProperties = (properties: Properties): Properties => {
  const sanitized: Properties = {};
  for (const [key, value] of Object.entries(properties)) {
    if (key === 'title') {
      // The page title names the resource being viewed.
      continue;
    }
    if (key === '$snapshot_data') {
      sanitized[key] = Array.isArray(value)
        ? value.map(sanitizeRecordingEntry)
        : sanitizeRecordingEntry(value);
    } else if (typeof value !== 'string') {
      sanitized[key] = value;
    } else if (isURL(value)) {
      sanitized[key] = sanitizeURL(value);
    } else if (key.endsWith('pathname')) {
      sanitized[key] = basePath() + routePattern(value);
    } else {
      sanitized[key] = value;
    }
  }
  return sanitized;
};

/**
 * sanitizeEvent reduces the URLs and paths PostHog attaches to every event,
 * session recordings included, to route patterns, and drops the page title.
 * Events are then grouped by the kind of page they happened on rather than
 * by which Project or Stage it showed, and query strings, which on the login
 * page can carry an OIDC authorization code, are never sent.
 *
 * It does not remove resource names from everything: what a user clicks and
 * what a session recording shows are sent as PostHog captures them.
 */
export const sanitizeEvent = (event: CaptureResult | null): CaptureResult | null =>
  event && {
    ...event,
    properties: sanitizeProperties(event.properties),
    ...(event.$set && { $set: sanitizeProperties(event.$set) }),
    ...(event.$set_once && { $set_once: sanitizeProperties(event.$set_once) })
  };
