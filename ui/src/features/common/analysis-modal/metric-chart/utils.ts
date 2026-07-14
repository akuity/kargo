/**
 * Formats a measurement value for display in the metric chart.
 *
 * Coalesces both null and undefined to an empty string. A data point's
 * chartValue object does not always carry every key in the deduped
 * conditionKeys union, so lookups such as `data.chartValue?.[cKey]` can be
 * undefined -- guarding only null (as an earlier version did) let
 * `undefined.toString()` throw.
 */
export const defaultValueFormatter = (value?: number | string | null): string =>
  value === null || value === undefined ? '' : value.toString();
