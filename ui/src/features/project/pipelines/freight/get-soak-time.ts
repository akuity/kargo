import { add, intervalToDuration, isBefore } from 'date-fns';

import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { Duration } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

// freight is in stage since X
// required soak time is Y duration
// add Y duration in X
// if it is in past, then freight is soaked
// if it is in future then difference between future time and current time is how much time it needs to soak
export const getSoakTime = (payload: {
  freight: Freight;
  // only relevant stages
  // for example,
  // in normal promotion stage-A -> stage-B, soak time for stage-B is decide upon how much time stage-A contains that freight
  // stage-B -> stage-C, stage-D, stage-E, soak time for stage-C, stage-D, stage-E is decide upon how much time stage-B contains that freight
  freightInStage: Stage;
  // time duration - 1h, 1h2m, 10m
  requiredSoakTime: Duration;
}) => {
  if (!payload.requiredSoakTime) {
    return '';
  }

  const sourceStageName = payload.freightInStage?.metadata?.name || '';

  const inSourceStageSince = timestampDate(
    payload.freight?.status?.currentlyIn?.[sourceStageName].since
  );

  if (!inSourceStageSince) {
    return '';
  }

  const requiredSoakTime = durationToHMS(payload.requiredSoakTime);

  const calculateSoakTimeSinceInFreight = add(inSourceStageSince, requiredSoakTime);

  const now = new Date();

  if (isBefore(now, calculateSoakTimeSinceInFreight)) {
    return intervalToDuration({ start: now, end: calculateSoakTimeSinceInFreight });
  }

  return null;
};

const durationToHMS = (duration: Duration) => {
  const totalSeconds = Number(duration.duration / 1_000_000_000n); // Convert to seconds

  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return { hours, minutes, seconds };
};
