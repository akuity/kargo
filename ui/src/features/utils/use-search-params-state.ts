import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { z, ZodArray, ZodObject, ZodRawShape, ZodRecord } from 'zod';

export const useSearchParamsState = <T extends ZodObject<ZodRawShape>>(schema: T) => {
  type StateType = z.infer<T>;

  const [search, setSearch] = useSearchParams();

  const state: StateType = useMemo(() => {
    const localState = {} as z.infer<T>;
    for (const getter of Object.keys(schema.shape)) {
      let base = schema.shape[getter];

      // get the basic type - z.string().required().transform()...
      //                          ^
      //                          |
      //                          |
      //                        we want to reach this definition
      // this type helps us decide how to parse search string
      while (base._def.innerType || base._def.schema) {
        base = base._def.innerType || base._def.schema;
      }

      const isArray = base instanceof ZodArray;

      const shouldTransformToJson = base instanceof ZodObject || base instanceof ZodRecord;

      // use the default parsing mechanism that user gave
      base = schema.shape[getter];

      let transformedStateValue;
      if (isArray) {
        transformedStateValue = base.safeParse(search.getAll(getter)).data;
      } else {
        let value = search.get(getter);
        if (shouldTransformToJson && value) {
          value = JSON.parse(value);
        }

        transformedStateValue = base.safeParse(value).data;
      }

      if (transformedStateValue !== undefined) {
        // @ts-expect-error getter is key of zod schema object and thats correct
        localState[getter] = transformedStateValue;
      }
    }

    return localState;
  }, [search]);

  const setSearchState = (nextState: Partial<StateType>) =>
    setSearch(
      (prevSearch) => {
        const newSearch = new URLSearchParams(prevSearch);

        // we will rebuild the value of "key"

        // delete keys first
        for (const [key, _value] of Object.entries(nextState)) {
          const typeSafeValue = _value as StateType[keyof StateType];

          newSearch.delete(key);

          let urlSafeValue: string;

          if (typeSafeValue === null) {
            // null is remove key operation
            continue;
          }

          // this doesn't support array of objects yet...
          // when there is requirement for array of objects
          // we simply want to perform zod instance check on this key in zod schema
          type UnsupportedArrayOfObjects = string[];

          if (Array.isArray(typeSafeValue)) {
            for (const arrValue of typeSafeValue as UnsupportedArrayOfObjects) {
              newSearch.append(key, arrValue);
            }
            continue;
          }

          if ((typeSafeValue as unknown) instanceof Date) {
            urlSafeValue = (typeSafeValue as Date).toISOString();
          } else if (typeof typeSafeValue === 'object') {
            urlSafeValue = JSON.stringify(typeSafeValue);
          } else {
            urlSafeValue = String(typeSafeValue);
          }

          if (typeof urlSafeValue === 'string') {
            newSearch.set(key, urlSafeValue);
            continue;
          }
        }

        return newSearch;
      },
      { preventScrollReset: true }
    );

  const removeKeysFromSearch = (key: string[]) =>
    setSearch(
      (prevSearch) => {
        const existingSearch = new URLSearchParams(prevSearch);

        for (const k of key) {
          existingSearch.delete(k);
        }

        return existingSearch;
      },
      { preventScrollReset: true }
    );

  return { state, setSearchState, removeKeysFromSearch };
};
