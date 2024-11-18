import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { Freight, FreightReference, Stage } from '@ui/gen/v1alpha1/generated_pb';

export const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';
export const DESCRIPTION_ANNOTATION_KEY = 'kargo.akuity.io/description';

export const getAlias = (freight?: Freight): string | undefined => {
  return freight?.alias || freight?.metadata?.labels[ALIAS_LABEL_KEY] || undefined;
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
    : [];
}

export function getCurrentFreightWarehouse(stage: Stage) {
  const freightRef = getCurrentFreight(stage);
  const isWarehouseKind = freightRef.reduce(
    (acc, cur) => acc || cur?.origin?.kind === 'Warehouse',
    false
  );

  if (isWarehouseKind) {
    return freightRef[0]?.origin?.name || stage?.spec?.requestedFreight[0]?.origin?.name || '';
  }

  return '';
}

export function currentFreightHasVerification(stage: Stage): boolean {
  const collection = stage?.status?.freightHistory[0];
  return (collection && (collection.verificationHistory || []).length > 0) || false;
}

export function mapToNames<T extends { metadata?: ObjectMeta }>(objects: T[]) {
  return (objects || []).reduce((acc, obj) => {
    if (obj?.metadata?.name) {
      acc.push(obj.metadata.name);
    }
    return acc;
  }, [] as string[]);
}
