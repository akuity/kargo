import { beforeEach, describe, expect, test, vi } from 'vitest';

const init = vi.fn();
const recorderLoaded = vi.fn();

vi.mock('posthog-js', () => ({ default: { init } }));
// The recorder registers itself as a side effect of being imported, so being
// imported is what there is to observe.
vi.mock('posthog-js/dist/posthog-recorder', () => {
  recorderLoaded();
  return {};
});

const config = {
  publicApiKey: 'phc_test',
  host: 'https://posthog.example.com',
  sessionReplayEnabled: false
};

describe('startPostHog', () => {
  beforeEach(() => {
    init.mockClear();
    recorderLoaded.mockClear();
    // The module remembers having started, so each test needs a fresh copy.
    vi.resetModules();
  });

  test('starts PostHog with the given key and host', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog(config);
    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith(
      'phc_test',
      expect.objectContaining({ api_host: 'https://posthog.example.com' })
    );
  });

  test('starts PostHog only once', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog(config);
    await startPostHog({ ...config, publicApiKey: 'phc_other' });
    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith('phc_test', expect.anything());
  });

  test('does not load the recorder unless session replay is enabled', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog(config);
    expect(recorderLoaded).not.toHaveBeenCalled();
    expect(init).toHaveBeenCalledWith(
      'phc_test',
      expect.objectContaining({ disable_session_recording: true })
    );
  });

  test('loads the recorder before starting when session replay is enabled', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog({ ...config, sessionReplayEnabled: true });
    expect(recorderLoaded).toHaveBeenCalledTimes(1);
    expect(recorderLoaded.mock.invocationCallOrder[0]).toBeLessThan(
      init.mock.invocationCallOrder[0]
    );
    expect(init).toHaveBeenCalledWith(
      'phc_test',
      expect.objectContaining({ disable_session_recording: false })
    );
  });
});

describe('postHogOptions', () => {
  test('reports page views only, anonymously and sanitized', async () => {
    const { postHogOptions } = await import('./posthog');
    // Imported here rather than at the top so that they are the same copies
    // the freshly imported module above holds.
    const { sanitizeEvent } = await import('./sanitize-event');
    const { sessionRecordingOptions } = await import('./session-recording');
    expect(postHogOptions(config)).toEqual({
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
      enable_recording_console_log: false,
      disable_session_recording: true,
      session_recording: sessionRecordingOptions,
      disable_external_dependency_loading: true
    });
  });

  test('records sessions, masked, when session replay is enabled', async () => {
    const { postHogOptions } = await import('./posthog');
    const { sessionRecordingOptions } = await import('./session-recording');
    const options = postHogOptions({ ...config, sessionReplayEnabled: true });
    expect(options.disable_session_recording).toBe(false);
    expect(options.session_recording).toBe(sessionRecordingOptions);
    // Recording brings none of the other capture back, and the recorder is
    // still not fetched from PostHog's host.
    expect(options.autocapture).toBe(false);
    expect(options.capture_performance).toBe(false);
    expect(options.enable_recording_console_log).toBe(false);
    expect(options.disable_external_dependency_loading).toBe(true);
  });
});
