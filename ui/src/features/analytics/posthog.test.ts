import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

const init = vi.fn();
const register = vi.fn();
const capture = vi.fn();

vi.mock('posthog-js', () => ({ default: { init, register, capture } }));

const config = {
  publicApiKey: 'phc_test',
  host: 'https://posthog.example.com',
  sessionReplayEnabled: false
};

const stubWindow = (base = '') =>
  vi.stubGlobal('window', {
    __KARGO_BASE_PATH__: base,
    location: { origin: 'https://kargo.example.com' }
  });

beforeEach(() => {
  init.mockClear();
  register.mockClear();
  capture.mockClear();
  // The module remembers having started, so each test needs a fresh copy.
  vi.resetModules();
  stubWindow();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('startPostHog', () => {
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

  test('attaches the instance URL to every event', async () => {
    const { startPostHog } = await import('./posthog');
    await startPostHog(config);
    expect(register).toHaveBeenCalledTimes(1);
    expect(register).toHaveBeenCalledWith({
      kargo_instance_url: 'https://kargo.example.com/'
    });
    expect(init.mock.invocationCallOrder[0]).toBeLessThan(register.mock.invocationCallOrder[0]);
  });
});

describe('instanceURL', () => {
  test('is the origin when the UI is served at the root', async () => {
    const { instanceURL } = await import('./posthog');
    expect(instanceURL()).toBe('https://kargo.example.com/');
  });

  test('includes the base path', async () => {
    stubWindow('/kargo');
    const { instanceURL } = await import('./posthog');
    expect(instanceURL()).toBe('https://kargo.example.com/kargo/');
  });

  test.for(['', '/kargo'])('survives sanitization with base path %j', async (base) => {
    stubWindow(base);
    const { instanceURL } = await import('./posthog');
    const { sanitizeEvent } = await import('./sanitize-event');
    const sanitized = sanitizeEvent({
      uuid: 'test-uuid',
      event: 'promotion_started',
      properties: { kargo_instance_url: instanceURL() }
    });
    expect(sanitized?.properties).toEqual({ kargo_instance_url: instanceURL() });
  });
});

describe('track', () => {
  test('does nothing before PostHog has started', async () => {
    const { track } = await import('./posthog');
    expect(() => track('promotion_started')).not.toThrow();
    expect(capture).not.toHaveBeenCalled();
  });

  test('reports an event once PostHog has started', async () => {
    const { startPostHog, track } = await import('./posthog');
    await startPostHog(config);
    track('promotion_started');
    expect(capture).toHaveBeenCalledWith('promotion_started', undefined);
  });

  test('reports an event with its properties', async () => {
    const { startPostHog, track } = await import('./posthog');
    await startPostHog(config);
    track('freight_approved', { source: 'timeline' });
    expect(capture).toHaveBeenCalledWith('freight_approved', { source: 'timeline' });
  });
});

describe('postHogOptions', () => {
  test('leaves PostHog as it is by default, bar what is set here', async () => {
    const { postHogOptions } = await import('./posthog');
    // Imported here rather than at the top so that it is the same copy the
    // freshly imported module above holds.
    const { sanitizeEvent } = await import('./sanitize-event');
    expect(postHogOptions(config)).toEqual({
      api_host: 'https://posthog.example.com',
      defaults: '2026-08-30',
      capture_pageview: 'history_change',
      before_send: sanitizeEvent,
      person_profiles: 'never',
      persistence: 'localStorage',
      respect_dnt: true,
      disable_session_recording: true,
      session_recording: { maskAllInputs: true }
    });
  });

  test('records sessions when session replay is enabled', async () => {
    const { postHogOptions } = await import('./posthog');
    const options = postHogOptions({ ...config, sessionReplayEnabled: true });
    expect(options.disable_session_recording).toBe(false);
    expect(options.session_recording).toEqual({ maskAllInputs: true });
  });
});
