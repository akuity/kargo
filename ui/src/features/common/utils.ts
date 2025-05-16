import { Freight, FreightReference, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { PlainMessageRecursive } from '@ui/utils/connectrpc-utils';

export const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';
export const DESCRIPTION_ANNOTATION_KEY = 'kargo.akuity.io/description';

export const getAlias = (freight?: PlainMessageRecursive<Freight>): string | undefined => {
  return freight?.alias || freight?.metadata?.labels?.[ALIAS_LABEL_KEY] || undefined;
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
  const isWarehouseKind = freightRef.some((freight) => freight?.origin?.kind === 'Warehouse');

  if (isWarehouseKind) {
    return freightRef[0]?.origin?.name || stage?.spec?.requestedFreight[0]?.origin?.name || '';
  }

  return '';
}

export function selectFreightByWarehouse(
  freightsInOrder /* they should be in order as if applied latest to old */ : Freight[],
  warehouse?: string
) {
  const LATEST_FREIGHT = 0;
  const order = freightsInOrder?.findIndex((freight) => freight?.origin?.name === warehouse);

  return order > -1 ? order : LATEST_FREIGHT;
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
