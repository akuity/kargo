import { Stage } from '@ui/gen/v1alpha1/generated_pb';

export const lastVerificationErrored = (stage: Stage): boolean => {
  const freightHistory = stage?.status?.freightHistory;
  if (!freightHistory || freightHistory.length === 0) {
    return false;
  }
  const verificationHistory = freightHistory[0].verificationHistory;
  if (!verificationHistory || verificationHistory.length === 0) {
    return false;
  }

  const lastVerification = verificationHistory[0];
  return (
    lastVerification.phase === 'Failed' ||
    lastVerification.phase === 'Error' ||
    lastVerification.phase === 'Inconclusive'
  );
};
