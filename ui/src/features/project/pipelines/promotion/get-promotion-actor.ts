import { Promotion } from '@ui/gen/api/v2/models';

export const getPromotionActor = (promotion: Promotion) => {
  const annotation = promotion?.metadata?.annotations?.['kargo.akuity.io/create-actor'];

  if (!annotation) {
    return 'N/A';
  }

  const separatorIndex = annotation.indexOf(':');

  return separatorIndex === -1 ? annotation : annotation.slice(separatorIndex + 1);
};
