import { useLayoutEffect, useMemo, useRef, useState } from 'react';

import { minDiscoveringMs } from '../nodes/warehouse-state';

// The names to show as discovering at `now`: those discovering now, and those
// whose discovery started less than minDiscoveringMs ago. `startedAt` records
// when each was first seen discovering, and is kept up to date in place.
export const holdDiscovering = (
  discovering: string[],
  startedAt: Record<string, number>,
  now: number
): { held: Set<string>; releaseAt?: number } => {
  const held = new Set(discovering);
  let releaseAt: number | undefined;

  for (const name of discovering) {
    startedAt[name] ??= now;
  }

  for (const [name, start] of Object.entries(startedAt)) {
    const end = start + minDiscoveringMs;

    if (held.has(name)) {
      continue;
    }

    if (now < end) {
      held.add(name);
      releaseAt = Math.min(releaseAt ?? end, end);
    } else {
      delete startedAt[name];
    }
  }

  return { held, releaseAt };
};

// holdDiscovering, worked out while rendering, so that a discovery starting or
// ending costs no render of its own. The one render it adds is when a hold
// runs out, which is when the loader and the edge animation have to stop.
export const useHeldDiscovering = (discovering: string[]): Set<string> => {
  const startedAt = useRef<Record<string, number>>({});
  const [released, setReleased] = useState(0);

  const { held, releaseAt } = holdDiscovering(discovering, startedAt.current, Date.now());

  // A layout effect, so that a timer made pointless by a render that has
  // released its hold already is cleared in that render's commit. A passive
  // effect clears it later, and the timer can fire in between for nothing.
  useLayoutEffect(() => {
    if (releaseAt === undefined) {
      return;
    }

    const timer = setTimeout(() => setReleased((n) => n + 1), releaseAt - Date.now());
    return () => clearTimeout(timer);
  }, [releaseAt, released]);

  // the same Set for as long as the names in it are the same
  const key = [...held].sort().join(',');
  return useMemo(() => held, [key]);
};
