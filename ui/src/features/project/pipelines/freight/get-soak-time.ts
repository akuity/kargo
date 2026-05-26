import { add, intervalToDuration, isBefore } from 'date-fns';

import { Freight, Stage } from '@ui/gen/api/v2/models';
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
  // time duration string e.g. 1h, 1h2m, 10m, 30s
  requiredSoakTime: string;
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

const durationToHMS = (duration: string) => {
  let totalSeconds = 0;
  const re = /([0-9]+(?:\.[0-9]+)?)(h|m|s)/g;
  let match;
  while ((match = re.exec(duration)) !== null) {
    const value = parseFloat(match[1]);
    switch (match[2]) {
      case 'h':
        totalSeconds += value * 3600;
        break;
      case 'm':
        totalSeconds += value * 60;
        break;
      case 's':
        totalSeconds += value;
        break;
    }
  }
  return {
    hours: Math.floor(totalSeconds / 3600),
    minutes: Math.floor((totalSeconds % 3600) / 60),
    seconds: Math.floor(totalSeconds % 60)
  };
};
