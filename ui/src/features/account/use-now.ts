import { useEffect, useState } from 'react';

// The current time, refreshed every `intervalMs`, so what a page says about
// it -- "in 3 minutes", "expired" -- keeps up while the page stays open.
export const useNow = (intervalMs: number) => {
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const interval = setInterval(() => setNow(new Date()), intervalMs);
    return () => clearInterval(interval);
  }, [intervalMs]);

  return now;
};
