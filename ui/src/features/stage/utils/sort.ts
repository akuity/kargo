import { Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const sortPromotions = (a: Promotion, b: Promotion) => {
  const nameDiff = (b.metadata?.name || '').localeCompare(a.metadata?.name || '');

  const aIsRunning = a.status?.phase === 'Running';
  const bIsRunning = b.status?.phase === 'Running';
  const aIsPending = a.status?.phase === 'Pending';
  const bIsPending = b.status?.phase === 'Pending';

  if (aIsRunning || bIsRunning) {
    if (aIsRunning && bIsRunning) {
      return nameDiff;
    }
    return aIsRunning ? -1 : 1;
  }

  if (aIsPending || bIsPending) {
    if (aIsPending && bIsPending) {
      return nameDiff;
    }
    return aIsPending ? -1 : 1;
  }

  return nameDiff;
};
