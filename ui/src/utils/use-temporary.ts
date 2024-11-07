// hook that temporary sets to true for given interval

import { useEffect, useState } from 'react';

// useful in scenarios where we'd want to show some message temporarily
export const useTemporaryBoolean = (timeout = 500 /* milliseconds */) => {
  const [init, setInit] = useState(false);

  useEffect(() => {
    if (init) {
      setTimeout(() => {
        setInit(false);
      }, timeout);
    }
  }, [init]);

  return [init, () => setInit(true)] as const;
};
