import { Freight, FreightReference, Stage } from '@ui/gen/v1alpha1/generated_pb';

export const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';
export const DESCRIPTION_ANNOTATION_KEY = 'kargo.akuity.io/description';

export const getAlias = (freight?: Freight): string | undefined => {
  return freight?.metadata?.labels[ALIAS_LABEL_KEY] || undefined;
};

export const dnsRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;

export interface HasDescriptionAnnotation {
  metadata?: {
    annotations?: {
      [DESCRIPTION_ANNOTATION_KEY]?: string;
    };
  };
}

export function getDescription<T extends HasDescriptionAnnotation>(item: T) {
  return item?.metadata?.annotations?.[DESCRIPTION_ANNOTATION_KEY];
}

export function getCurrentFreight(stage: Stage): FreightReference[] {
  return stage?.status?.freightHistory[0]
    ? Object.values(stage?.status?.freightHistory[0]?.items)
    : stage?.status?.currentFreight
      ? [stage?.status?.currentFreight]
      : [];
}

export function currentFreightHasVerification(stage: Stage): boolean {
  const collection = stage?.status?.freightHistory[0];
  if (
    (collection && (collection.verificationHistory || []).length > 0) ||
    stage?.status?.currentFreight?.verificationHistory
  ) {
    return true;
  }
  return false;
}
