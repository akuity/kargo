import type { CaptureResult } from 'posthog-js';
import { afterEach, describe, expect, test, vi } from 'vitest';

import { routePattern, sanitizeEvent } from './sanitize-event';

const origin = 'https://kargo.example.com';

const stubWindow = (base = '') =>
  vi.stubGlobal('window', { __KARGO_BASE_PATH__: base, location: { origin } });

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('routePattern', () => {
  test.for([
    ['/', '/'],
    ['/login', '/login'],
    ['/project/my-project', '/project/:name'],
    ['/project/my-project/', '/project/:name'],
    ['/project/my-project/stage/prod', '/project/:name/stage/:stageName'],
    ['/project/my-project/create-stage', '/project/:name/create-stage'],
    ['/project/my-project/freight/abc123', '/project/:name/freight/:freightName'],
    ['/project/my-project/warehouse/images', '/project/:name/warehouse/:warehouseName/:tab?'],
    ['/project/my-project/warehouse/images/yaml', '/project/:name/warehouse/:warehouseName/:tab?'],
    [
      '/project/my-project/promote/freight/abc123/stage/prod',
      '/project/:name/promote/freight/:freight/stage/:stage'
    ],
    [
      '/ext/project/my-project/stage/prod/analysis-run/run-1/logs',
      '/ext/project/:name/stage/:stageName/analysis-run/:analysisRunId/logs'
    ],
    // Paths beneath a known route fall back to that route's wildcard.
    ['/project/my-project/settings/roles', '/project/:name/settings/*'],
    ['/project/my-project/ext/my-extension', '/project/:name/ext/*'],
    ['/settings/cluster-secrets', '/settings/*'],
    ['/project/my-project/unknown/thing', '/project/:name/*'],
    // Paths no route knows fall back to the root wildcard.
    ['/unknown/my-secret-name', '/*']
  ])('turns %j into %j', ([pathname, expected]) => {
    stubWindow();
    expect(routePattern(pathname)).toBe(expected);
  });

  test('strips the base path before matching', () => {
    stubWindow('/kargo');
    expect(routePattern('/kargo/project/my-project')).toBe('/project/:name');
  });

  test('does not match a path outside the base path', () => {
    stubWindow('/kargo');
    expect(routePattern('/project/my-project')).toBe('/*');
  });
});

describe('sanitizeEvent', () => {
  const event = (overrides: Partial<CaptureResult>): CaptureResult => ({
    uuid: 'test-uuid',
    event: '$pageview',
    properties: {},
    ...overrides
  });

  test('passes null through', () => {
    stubWindow();
    expect(sanitizeEvent(null)).toBeNull();
  });

  test('replaces URLs and paths with route patterns', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({
        properties: {
          $current_url: `${origin}/project/my-project/stage/prod?tab=verifications#top`,
          $pathname: '/project/my-project/stage/prod',
          $prev_pageview_pathname: '/project/my-project'
        }
      })
    );
    expect(sanitized?.properties).toEqual({
      $current_url: `${origin}/project/:name/stage/:stageName`,
      $pathname: '/project/:name/stage/:stageName',
      $prev_pageview_pathname: '/project/:name'
    });
  });

  test('drops the query string of the login page', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({ properties: { $current_url: `${origin}/login?code=secret&state=xyz` } })
    );
    expect(sanitized?.properties).toEqual({ $current_url: `${origin}/login` });
  });

  test('keeps the base path in what it reports', () => {
    stubWindow('/kargo');
    const sanitized = sanitizeEvent(
      event({
        properties: {
          $current_url: `${origin}/kargo/project/my-project`,
          $pathname: '/kargo/project/my-project'
        }
      })
    );
    expect(sanitized?.properties).toEqual({
      $current_url: `${origin}/kargo/project/:name`,
      $pathname: '/kargo/project/:name'
    });
  });

  test('reduces URLs of other origins to the origin', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({
        properties: {
          $referrer: 'https://idp.example.com/auth?client_id=kargo&state=xyz',
          $referring_domain: 'idp.example.com'
        }
      })
    );
    expect(sanitized?.properties).toEqual({
      $referrer: 'https://idp.example.com',
      $referring_domain: 'idp.example.com'
    });
  });

  test('removes the page title', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({ properties: { title: 'Stage: prod - my-project - Kargo' } })
    );
    expect(sanitized?.properties).toEqual({});
  });

  test('leaves other properties alone', () => {
    stubWindow();
    const properties = {
      $browser: 'Firefox',
      $referrer: '$direct',
      $screen_height: 1080,
      $is_identified: false
    };
    expect(sanitizeEvent(event({ properties }))?.properties).toEqual(properties);
  });

  test('sanitizes person properties', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({
        $set: { $current_url: `${origin}/project/my-project` },
        $set_once: {
          $initial_current_url: `${origin}/project/my-project/freight/abc123`,
          $initial_pathname: '/project/my-project/freight/abc123'
        }
      })
    );
    expect(sanitized?.$set).toEqual({ $current_url: `${origin}/project/:name` });
    expect(sanitized?.$set_once).toEqual({
      $initial_current_url: `${origin}/project/:name/freight/:freightName`,
      $initial_pathname: '/project/:name/freight/:freightName'
    });
  });

  test('sanitizes the page URL inside a session recording', () => {
    stubWindow();
    const fullSnapshot = { type: 2, data: 'compressed', cv: '2024-10', timestamp: 1 };
    const mutation = { type: 3, data: { source: 0, texts: 'compressed' }, timestamp: 2 };
    const sanitized = sanitizeEvent(
      event({
        event: '$snapshot',
        properties: {
          $session_id: 'test-session',
          $snapshot_data: [
            {
              type: 4,
              data: { href: `${origin}/project/my-project?tab=x`, width: 1024, height: 768 },
              timestamp: 0
            },
            fullSnapshot,
            mutation,
            {
              type: 5,
              data: {
                tag: '$url_changed',
                payload: { href: `${origin}/project/my-project/stage/prod` }
              },
              timestamp: 3
            },
            { type: 5, data: { tag: '$recording_started', payload: { reason: 'x' } }, timestamp: 4 }
          ]
        }
      })
    );
    expect(sanitized?.properties).toEqual({
      $session_id: 'test-session',
      $snapshot_data: [
        {
          type: 4,
          data: { href: `${origin}/project/:name`, width: 1024, height: 768 },
          timestamp: 0
        },
        fullSnapshot,
        mutation,
        {
          type: 5,
          data: {
            tag: '$url_changed',
            payload: { href: `${origin}/project/:name/stage/:stageName` }
          },
          timestamp: 3
        },
        { type: 5, data: { tag: '$recording_started', payload: { reason: 'x' } }, timestamp: 4 }
      ]
    });
  });

  test('sanitizes a session recording sent as a single entry', () => {
    stubWindow();
    const sanitized = sanitizeEvent(
      event({
        event: '$snapshot',
        properties: {
          $snapshot_data: { type: 4, data: { href: `${origin}/project/my-project` } }
        }
      })
    );
    expect(sanitized?.properties.$snapshot_data).toEqual({
      type: 4,
      data: { href: `${origin}/project/:name` }
    });
  });

  test('does not add person properties an event lacks', () => {
    stubWindow();
    const sanitized = sanitizeEvent(event({}));
    expect(sanitized).not.toHaveProperty('$set');
    expect(sanitized).not.toHaveProperty('$set_once');
  });
});
