// remove nested empty array
// remove nested empty object
// remove nullish/undefined values
// read helpers.test.ts for examples
export const cleanEmptyObjectValues = <T extends Record<string, unknown>>(obj: T): T => {
  obj = obj || {};

  //   recursively remove nested empty
  for (const [k, v] of Object.entries(obj)) {
    if (Array.isArray(v)) {
      // @ts-expect-error
      obj[k] = v.filter((element) => {
        if (typeof element === 'object') {
          return Object.keys(cleanEmptyObjectValues(element)).length > 0;
        }

        return element !== null && element !== undefined;
      });
    }

    if (typeof v === 'object') {
      // @ts-expect-error
      obj[k] = cleanEmptyObjectValues(v);
    }
  }

  for (const [k, v] of Object.entries(obj)) {
    if (
      v === null ||
      v === undefined ||
      (Array.isArray(v) && v.length === 0) ||
      (typeof v === 'object' && Object.keys(v).length === 0)
    ) {
      delete obj[k];
    }
  }

  return obj;
};

export const removePropertiesRecursively = <T extends Record<string, unknown>>(
  schema: T,
  props: string[]
) => {
  // remove keys
  for (const prop of props) {
    if (schema?.[prop]) {
      delete schema[prop];
    }
  }

  // recurse
  for (const [key, value] of Object.entries(schema || {})) {
    if (typeof value === 'object') {
      schema[key as keyof T] = removePropertiesRecursively(
        value as Record<string, unknown>,
        props
      ) as T[keyof T];
    }
  }

  return schema;
};
