import { beforeEach, describe, expect, test, vi } from 'vitest';

const init = vi.fn();

vi.mock('posthog-js', () => ({ default: { init } }));

describe('startPostHog', () => {
  beforeEach(() => {
    init.mockClear();
    // The module remembers having started, so each test needs a fresh copy.
    vi.resetModules();
  });

  test('starts PostHog with the given key and host', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog('phc_test', 'https://posthog.example.com');
    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith(
      'phc_test',
      expect.objectContaining({ api_host: 'https://posthog.example.com' })
    );
  });

  test('starts PostHog only once', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog('phc_test', 'https://posthog.example.com');
    await startPostHog('phc_other', 'https://other.example.com');
    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith('phc_test', expect.anything());
  });
});

describe('postHogOptions', () => {
  test('reports page views only, anonymously and sanitized', async () => {
    const { postHogOptions } = await import('./posthog');
    // Imported here rather than at the top so that it is the same copy the
    // freshly imported module above holds.
    const { sanitizeEvent } = await import('./sanitize-event');
    expect(postHogOptions('https://posthog.example.com')).toEqual({
      api_host: 'https://posthog.example.com',
      capture_pageview: 'history_change',
      before_send: sanitizeEvent,
      person_profiles: 'never',
      persistence: 'localStorage',
      respect_dnt: true,
      autocapture: false,
      rageclick: false,
      capture_heatmaps: false,
      capture_dead_clicks: false,
      capture_exceptions: false,
      capture_performance: false,
      disable_session_recording: true,
      disable_external_dependency_loading: true
    });
  });
});
