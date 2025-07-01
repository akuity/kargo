import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

export const getPromotionActor = (promotion: Promotion) => {
  const annotation = promotion?.metadata?.annotations?.['kargo.akuity.io/create-actor'];

  const email = annotation?.split(':')[1];

  return email || annotation || 'N/A';
};
