import type { PostHogConfig } from 'posthog-js';

import { sanitizeEvent } from './sanitize-event';
import { sessionRecordingOptions } from './session-recording';

export type AnalyticsConfig = {
  publicApiKey: string;
  host: string;
  sessionReplayEnabled: boolean;
};

/**
 * postHogOptions returns the options PostHog is started with. They limit what
 * is reported to page views and, when an operator has enabled it, masked
 * session recordings. Users are not identified, and resource names are
 * removed from everything that is sent.
 */
export const postHogOptions = ({
  host,
  sessionReplayEnabled
}: Pick<AnalyticsConfig, 'host' | 'sessionReplayEnabled'>): Partial<PostHogConfig> => ({
  api_host: host,
  // The UI is a single-page app: a page view is a change of route.
  capture_pageview: 'history_change',
  before_send: sanitizeEvent,
  // Users are anonymous. Their random ID is kept in localStorage rather than
  // in a cookie, which PostHog would share with sibling subdomains.
  person_profiles: 'never',
  persistence: 'localStorage',
  respect_dnt: true,
  // Each of these would capture what is on screen or what the user does
  // with it, and with that the names and contents of resources.
  autocapture: false,
  rageclick: false,
  capture_heatmaps: false,
  capture_dead_clicks: false,
  capture_exceptions: false,
  capture_performance: false,
  enable_recording_console_log: false,
  // Session replay is recorded only when an operator has asked for it, and
  // then only as masked by sessionRecordingOptions.
  disable_session_recording: !sessionReplayEnabled,
  session_recording: sessionRecordingOptions,
  // PostHog loads further features (surveys, and so on) as scripts from its
  // own host when they are enabled in the PostHog project. Refusing those
  // scripts keeps such features off whatever the project's settings say.
  disable_external_dependency_loading: true
});

let started = false;

/**
 * startPostHog loads PostHog and starts reporting to it. PostHog is loaded on
 * demand so that a UI without analytics enabled never downloads it, and its
 * session recorder only when session replay is enabled. Calls after the first
 * do nothing.
 */
export const startPostHog = async (config: AnalyticsConfig): Promise<void> => {
  if (started) {
    return;
  }
  started = true;
  const { default: posthog } = await import('posthog-js');
  if (config.sessionReplayEnabled) {
    // The recorder is one of the scripts PostHog would otherwise load from
    // its own host. Importing it makes it part of the UI's build instead.
    await import('posthog-js/dist/posthog-recorder');
  }
  posthog.init(config.publicApiKey, postHogOptions(config));
};
