import { V1ObjectMeta } from '@ui/gen/api/v2/models';

// getPromotionActor names whoever asked for a promotion. Promotions and
// PromotionRollouts both record it in the same annotation, so one reader serves
// both; a promotion the control plane started on its own carries none.
export const getPromotionActor = (resource: { metadata?: V1ObjectMeta }) => {
  const annotation = resource?.metadata?.annotations?.['kargo.akuity.io/create-actor'];

  if (!annotation) {
    return 'N/A';
  }

  const separatorIndex = annotation.indexOf(':');

  return separatorIndex === -1 ? annotation : annotation.slice(separatorIndex + 1);
};
