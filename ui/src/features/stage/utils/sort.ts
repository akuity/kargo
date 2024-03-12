import { Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const sortPromotions = (a: Promotion, b: Promotion) => {
  const timestampDiff =
    Number(b.metadata?.creationTimestamp?.seconds || 0) -
    Number(a.metadata?.creationTimestamp?.seconds || 0);

  const aIsRunning = a.status?.phase === 'Running';
  const bIsRunning = b.status?.phase === 'Running';
  const aIsPending = a.status?.phase === 'Pending';
  const bIsPending = b.status?.phase === 'Pending';

  if (aIsRunning || bIsRunning) {
    if (aIsRunning && bIsRunning) {
      return timestampDiff;
    }
    return aIsRunning ? -1 : 1;
  }

  if (aIsPending || bIsPending) {
    if (aIsPending && bIsPending) {
      return timestampDiff;
    }
    return aIsPending ? -1 : 1;
  }

  return timestampDiff;
};
