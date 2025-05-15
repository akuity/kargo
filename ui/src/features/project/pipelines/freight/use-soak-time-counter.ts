import { Duration } from 'date-fns';
import { useEffect, useMemo, useState } from 'react';

import { queryCache } from '@ui/features/utils/cache';

import { durationToSeconds } from './duration-to-seconds';

export const useSoakTimeCounter = (soakTime?: Duration) => {
  const [counter, setCounter] = useState(soakTime);

  useEffect(() => {
    const interval = setInterval(() => {
      setCounter((prev) => {
        if (!prev) {
          return prev;
        }

        const newCounter = { ...prev };
        if (newCounter.seconds !== undefined) {
          newCounter.seconds -= 1;
        }

        if ((newCounter?.seconds || 0) < 0) {
          newCounter.seconds = 59;

          if (newCounter.minutes !== undefined) {
            newCounter.minutes -= 1;
          }
        }

        if ((newCounter?.minutes || 0) < 0) {
          newCounter.minutes = 59;

          if (newCounter.hours !== undefined) {
            newCounter.hours -= 1;
          }
        }

        if ((prev?.seconds || 0) === 0 && (prev?.minutes || 0) === 0 && (prev?.hours || 0) === 0) {
          clearInterval(interval);
          queryCache.freight.refetchQueryFreight();
          return undefined;
        }

        return newCounter;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    setCounter(soakTime);
  }, [soakTime]);

  return counter;
};

export const useSoakTimePercentage = (initialSoakTime?: Duration, soakTimeLeft?: Duration) =>
  useMemo(() => {
    if (!initialSoakTime || !soakTimeLeft) {
      return 0;
    }

    const initialSoakTimeSeconds = durationToSeconds(initialSoakTime);
    const soakTimeLeftSeconds = durationToSeconds(soakTimeLeft);

    return (soakTimeLeftSeconds * 100) / initialSoakTimeSeconds;
  }, [initialSoakTime, soakTimeLeft]);
