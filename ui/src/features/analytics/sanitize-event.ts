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

const sanitizeProperties = (properties: Properties): Properties => {
  const sanitized: Properties = {};
  for (const [key, value] of Object.entries(properties)) {
    if (key === 'title') {
      // The page title names the resource being viewed.
      continue;
    }
    if (typeof value !== 'string') {
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
 * sanitizeEvent removes the names of Kargo resources from an event before it
 * is sent to PostHog. They appear in the URLs, paths, and page titles that
 * PostHog attaches to every event.
 */
export const sanitizeEvent = (event: CaptureResult | null): CaptureResult | null =>
  event && {
    ...event,
    properties: sanitizeProperties(event.properties),
    ...(event.$set && { $set: sanitizeProperties(event.$set) }),
    ...(event.$set_once && { $set_once: sanitizeProperties(event.$set_once) })
  };
