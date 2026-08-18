import { useEffect, useState } from 'react';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import { Promotion, StepExecutionMetadata } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { elapsedStepDurationMs, formatStepDuration } from './utils/step-duration';

const TICK_INTERVAL_MS = 1000;

const RunningStepDuration = ({ meta }: { meta?: StepExecutionMetadata }) => {
  const [elapsed, setElapsed] = useState(() => elapsedStepDurationMs(meta));

  useEffect(() => {
    const interval = setInterval(() => setElapsed(elapsedStepDurationMs(meta)), TICK_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [meta]);

  if (elapsed === null) {
    return null;
  }

  return (
    <span
      className={'text-xs text-gray-500 ml-auto tabular-nums'}
      title={`Started ${meta?.startedAt}`}
    >
      {formatStepDuration(elapsed)}
    </span>
  );
};

export const StepDuration = ({
  promotion,
  stepIndex
}: {
  promotion?: Promotion;
  stepIndex: number;
}) => {
  const meta = promotion?.status?.stepExecutionMetadata?.[stepIndex];

  if (
    getPromotionDirectiveStepStatus(stepIndex, promotion?.status) ===
    PromotionDirectiveStepStatus.RUNNING
  ) {
    return <RunningStepDuration meta={meta} />;
  }

  const startedAt = parseDate(meta?.startedAt);
  // A Promotion aborted or errored mid-step can leave that step's finishedAt
  // unstamped. The Promotion's own finishedAt is when the step really stopped.
  const finishedAtRaw = meta?.finishedAt || promotion?.status?.finishedAt;
  const finishedAt = parseDate(finishedAtRaw);

  if (!startedAt || !finishedAt) {
    return null;
  }

  return (
    <span
      className={'text-xs text-gray-500 ml-auto tabular-nums'}
      title={`Started ${meta?.startedAt}\nFinished ${finishedAtRaw}`}
    >
      {formatStepDuration(finishedAt.getTime() - startedAt.getTime())}
    </span>
  );
};
