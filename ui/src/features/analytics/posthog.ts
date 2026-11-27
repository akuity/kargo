import type { PostHog, PostHogConfig, Properties } from 'posthog-js';

import { basePath } from '@ui/config/base-path';

import { sanitizeEvent } from './sanitize-event';

export type AnalyticsConfig = {
  publicApiKey: string;
  host: string;
  sessionReplayEnabled: boolean;
};

/**
 * postHogOptions returns the options PostHog is started with. Beyond these,
 * PostHog behaves as it does by default: it captures page views and what
 * users click, and whatever else is switched on in the PostHog project
 * (heatmaps, web vitals, error tracking, surveys, and so on).
 */
export const postHogOptions = ({
  host,
  sessionReplayEnabled
}: Pick<AnalyticsConfig, 'host' | 'sessionReplayEnabled'>): Partial<PostHogConfig> => ({
  api_host: host,
  // Opt in to PostHog's current defaults rather than its legacy ones.
  defaults: '2026-08-30',
  // The UI is a single-page app: a page view is a change of route.
  capture_pageview: 'history_change',
  before_send: sanitizeEvent,
  // Users are anonymous. Their random ID is kept in localStorage rather than
  // in a cookie, which PostHog would share with sibling subdomains.
  person_profiles: 'never',
  persistence: 'localStorage',
  respect_dnt: true,
  // Sessions are recorded only when an operator has asked for it, and then
  // as the PostHog project's replay settings say, with one exception: what
  // users type is always masked, since the UI's forms take credentials.
  disable_session_recording: !sessionReplayEnabled,
  session_recording: { maskAllInputs: true }
});

/**
 * instanceURL returns the URL this Kargo instance's UI is served at. It is
 * attached to every event so that events from different instances reporting
 * to one PostHog project can be told apart.
 */
export const instanceURL = (): string => `${window.location.origin}${basePath()}/`;

let started = false;
let client: PostHog | undefined;

/**
 * startPostHog loads PostHog and starts reporting to it. PostHog is loaded on
 * demand so that a UI without analytics enabled never downloads it. Calls
 * after the first do nothing.
 */
export const startPostHog = async (config: AnalyticsConfig): Promise<void> => {
  if (started) {
    return;
  }
  started = true;
  const { default: posthog } = await import('posthog-js');
  posthog.init(config.publicApiKey, postHogOptions(config));
  posthog.register({ kargo_instance_url: instanceURL() });
  client = posthog;
};

/**
 * track reports that something happened, for example that a button was
 * pressed:
 *
 *   <Button onClick={() => { track('promotion_started'); promote(); }} />
 *
 * It does nothing unless an operator has enabled analytics, so callers need
 * not check. Name events for what the user did, in snake_case. Properties are
 * sent as given, except that URLs among them are reduced to route patterns
 * like any other. Leave the names of Projects, Stages, and other resources
 * out of them unless they are meant to be reported.
 */
export const track = (event: string, properties?: Properties): void => {
  client?.capture(event, properties);
};
