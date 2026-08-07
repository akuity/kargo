/**
 * parseDate converts an API timestamp string (ISO 8601) into a Date, returning
 * undefined for a missing or unparseable value.
 *
 * REST timestamps are optional strings, and `new Date('')` / `new Date(undefined)`
 * produce a *truthy* Invalid Date. Callers that guard with `date ? ... : ...`
 * must use this instead of constructing a Date inline -- otherwise the guard
 * always passes and date-fns `format`/`formatDistance` throw "Invalid time
 * value" on the invalid date.
 */
export const parseDate = (timestamp?: string): Date | undefined => {
  if (!timestamp) {
    return undefined;
  }
  const date = new Date(timestamp);
  return isNaN(date.getTime()) ? undefined : date;
};
