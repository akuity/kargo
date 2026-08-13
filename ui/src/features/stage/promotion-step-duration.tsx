import { useEffect, useState } from 'react';

import { StepExecutionMetadata } from '@ui/gen/api/v2/models';

import { formatStepDuration, stepDurationMs } from './utils/step-duration';

// A running step re-renders on this interval so its elapsed time counts up,
// which is what distinguishes a step that is stuck from one that is merely slow.
const TICK_INTERVAL_MS = 1000;

export const StepDuration = ({ meta }: { meta?: StepExecutionMetadata }) => {
  const running = !!meta?.startedAt && !meta?.finishedAt;

  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!running) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), TICK_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [running]);

  const elapsed = stepDurationMs(meta, now);

  if (elapsed === null) {
    return null;
  }

  return (
    <span
      className='text-xs text-gray-500 ml-auto tabular-nums'
      title={
        meta?.finishedAt
          ? `Started ${meta.startedAt}, finished ${meta.finishedAt}`
          : `Started ${meta?.startedAt}`
      }
    >
      {formatStepDuration(elapsed)}
    </span>
  );
};
