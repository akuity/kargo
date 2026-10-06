import { FormatDistanceToken, formatDuration, intervalToDuration } from 'date-fns';

import { StepExecutionMetadata } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

const oneSecondMs = 1000;
const oneHourMs = 60 * 60 * oneSecondMs;

// formatDuration delegates each unit to locale.formatDistance(token, count),
// which is the supported hook for non-English output. Reusing it for compact
// units keeps date-fns in charge of picking and normalizing units while
// rendering "1h 40m" instead of en-US's "1 hour 40 minutes".
const compactUnits: Partial<Record<FormatDistanceToken, string>> = {
  xSeconds: 's',
  xMinutes: 'm',
  xHours: 'h',
  xDays: 'd',
  xMonths: 'mo',
  xYears: 'y'
};

const compactLocale = {
  formatDistance: (token: FormatDistanceToken, count: number) =>
    `${count}${compactUnits[token] ?? ''}`
};

/**
 * elapsedStepDurationMs returns how long a still-running promotion step has
 * been executing, in milliseconds, measured from startedAt to `now`.
 *
 * Callers pass a `now` that advances on a timer so the elapsed time counts up.
 * Returns null when the step has not started yet or startedAt is unparseable.
 */
export const elapsedStepDurationMs = (
  meta?: StepExecutionMetadata,
  now: number = Date.now()
): number | null => {
  const startedAt = parseDate(meta?.startedAt);
  if (!startedAt) {
    return null;
  }

  const elapsed = now - startedAt.getTime();
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
  // intervalToDuration truncates to whole seconds, so a sub-second value would
  // arrive at formatDuration as an empty Duration and render as ''. API
  // timestamps are RFC 3339 with second precision, so exactly 0 is the common
  // case, and "0ms" would imply a precision the data does not have. Other
  // sub-second values are still formatted exactly, in case finer-grained
  // timestamps arrive later.
  if (ms < oneSecondMs) {
    return ms === 0 ? '<1s' : `${ms}ms`;
  }

  return formatDuration(intervalToDuration({ start: 0, end: ms }), {
    locale: compactLocale,
    // Seconds are meaningful precision below an hour and noise above it. Every
    // coarser unit is listed so a long duration never silently loses its
    // largest component -- intervalToDuration rolls 40 days up into
    // {months: 1, days: 9}, which a shorter list would render as just "9d".
    format:
      ms < oneHourMs ? ['minutes', 'seconds'] : ['years', 'months', 'days', 'hours', 'minutes']
  });
};
