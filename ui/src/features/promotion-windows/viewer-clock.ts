const MINUTE_MS = 60_000;

export const toViewerClockMs = (instantMs: number) =>
  instantMs - new Date(instantMs).getTimezoneOffset() * MINUTE_MS;

export const toViewerClockDate = (instant: Date) => new Date(toViewerClockMs(instant.getTime()));

export const fromViewerClockDate = (viewerClock: Date) =>
  new Date(viewerClock.getTime() + viewerClock.getTimezoneOffset() * MINUTE_MS);
