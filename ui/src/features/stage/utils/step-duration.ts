import { StepExecutionMetadata } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

/**
 * stepDurationMs returns how long a promotion step has been executing, in
 * milliseconds.
 *
 * A step that has finished is measured from startedAt to finishedAt. A step
 * that is still running is measured from startedAt to `now`, so callers that
 * re-render on a timer show a live elapsed time. Returns null when the step has
 * not started yet, or when the timestamps are missing or unparseable.
 */
export const stepDurationMs = (
  meta?: StepExecutionMetadata,
  now: number = Date.now()
): number | null => {
  const startedAt = parseDate(meta?.startedAt);
  if (!startedAt) {
    return null;
  }

  const finishedAt = parseDate(meta?.finishedAt);
  const end = finishedAt ? finishedAt.getTime() : now;

  const elapsed = end - startedAt.getTime();
  // Clock skew between the control plane and the browser can put `now` behind
  // startedAt. Showing "-3s" would be worse than showing nothing.
  return elapsed < 0 ? null : elapsed;
};

/**
 * formatStepDuration renders a duration in milliseconds compactly.
 *
 * Step durations span many orders of magnitude -- a yaml-update finishes in
 * milliseconds while a git-wait-for-pr may run for hours -- so the unit adapts
 * rather than using a single fixed granularity.
 */
export const formatStepDuration = (ms: number): string => {
  // API timestamps are RFC 3339 with second precision, so any step faster than
  // a second computes to exactly 0. Rendering that as "0ms" would imply a
  // precision the data does not have. Sub-second values other than 0 are still
  // formatted exactly, in case finer-grained timestamps arrive later.
  if (ms === 0) {
    return '<1s';
  }

  if (ms < 1000) {
    return `${ms}ms`;
  }

  const totalSeconds = Math.floor(ms / 1000);
  if (totalSeconds < 60) {
    return `${totalSeconds}s`;
  }

  const totalMinutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (totalMinutes < 60) {
    return seconds > 0 ? `${totalMinutes}m ${seconds}s` : `${totalMinutes}m`;
  }

  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
};
