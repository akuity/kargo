/**
 * Query parameter serializer for the orval-generated API client.
 *
 * Array-valued query parameters must be sent as repeated keys
 * (`?freightOrigins=a&freightOrigins=b`), which is what the Kargo API expects.
 *
 * swagger.json declares these parameters as `collectionFormat: csv` because
 * that is swag's default, not because the server parses comma-joined values.
 * Orval honors the spec, so without this serializer an array would be sent as
 * a single comma-joined value, and -- worse -- an *empty* array would be sent
 * as an empty value (`?freightOrigins=`), which the server reads as a filter
 * on an origin whose name is the empty string, matching nothing.
 *
 * Omitting an empty array entirely is therefore deliberate: "no filter" must
 * send no parameter at all.
 */
export const serializeParams = (params?: Record<string, unknown>): string => {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params ?? {})) {
    if (value === undefined) {
      continue;
    }

    // An empty array appends nothing, so "no filter" sends no parameter.
    if (Array.isArray(value)) {
      for (const item of value) {
        searchParams.append(key, item === null ? 'null' : String(item));
      }
      continue;
    }

    searchParams.append(key, value === null ? 'null' : String(value));
  }

  return searchParams.toString();
};

export default serializeParams;
