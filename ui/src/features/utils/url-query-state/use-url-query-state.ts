// generic solution to use URL query as a state

import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

type SimpleState = Record<string, string>; /* lets keep it simple string value for now */

export const useURLQueryState = <T extends Record<string, unknown>>() => {
  const [search, setSearch] = useSearchParams();

  const state = useMemo(() => {
    return search.entries().reduce(
      (prev, entry) => {
        const [key, value] = entry;
        // @ts-expect-error strange error
        prev[key] = value;
        return prev;
      },
      {} as Partial<T & SimpleState>
    );
  }, [search]);

  return [
    state,
    (nextState: Partial<T & SimpleState> = {}) => {
      const nextSearch = new URLSearchParams();

      for (const [key, value] of Object.entries(nextState)) {
        nextSearch.set(key, value);
      }

      setSearch(nextSearch);
    }
  ] as const;
};
