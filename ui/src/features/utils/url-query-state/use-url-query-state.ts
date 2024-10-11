// generic solution to use URL query as a state

import queryString from 'query-string';
import { useCallback, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

type SimpleState = Record<string, string>; /* lets keep it simple string value for now */

export const useURLQueryState = <T extends Record<string, unknown>>() => {
  const [search] = useSearchParams();
  const navigate = useNavigate();

  const state = useMemo(() => queryString.parse(search.toString()), [search]);

  const setState = useCallback(
    (
      _nextState: Partial<T & SimpleState> = {},
      _opts?: { replace?: boolean; overwriteState?: boolean }
    ) => {
      // put default opts here
      const opts = {
        replace: Boolean(_opts?.replace),
        overwriteState: Boolean(_opts?.overwriteState)
      };

      let nextState = {};

      if (opts.overwriteState) {
        nextState = _nextState;
      } else {
        nextState = {
          ...state,
          ..._nextState
        };
      }

      navigate({ search: queryString.stringify(nextState) }, { replace: opts.replace });
    },
    [state]
  );

  const clearState = useCallback(() => setState({}, { overwriteState: true }), []);

  return [state as T, setState, clearState] as const;
};
