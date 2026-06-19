import { Freight, FreightReference, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { PlainMessageRecursive } from '@ui/utils/connectrpc-utils';

export const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';
export const DESCRIPTION_ANNOTATION_KEY = 'kargo.akuity.io/description';
export const REPLICATE_TO_ANNOTATION_KEY = 'kargo.akuity.io/replicate-to';
export const REPLICATE_TO_ALL_VALUE = '*';

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

// getCurrentFreightForComparison returns the piece of Freight currently on the
// Stage that originates from the same Warehouse as the incoming Freight, so a
// promotion comparison pairs like-for-like artifacts. A Stage in a
// multi-Warehouse project holds one piece of Freight per Warehouse; comparing
// the incoming Freight against an arbitrary one (e.g. the first) would mislabel
// every artifact as changed or new. Returns undefined when the Stage has no
// Freight from the incoming Freight's origin.
export function getCurrentFreightForComparison(
  stage: Stage,
  incoming?: Freight
): FreightReference | undefined {
  const originName = incoming?.origin?.name;
  return getCurrentFreight(stage).find((freight) => freight?.origin?.name === originName);
}

export interface CurrentFreightItem {
  reference: FreightReference;
  alias?: string;
}

export function getCurrentFreightByWarehouse(
  stage: Stage,
  freightMap?: Record<string, Freight>
): Record<string, CurrentFreightItem> {
  const items = stage?.status?.freightHistory?.[0]?.items || {};

  const result: Record<string, CurrentFreightItem> = {};
  for (const [warehouseIdentifier, reference] of Object.entries(items)) {
    const fullFreight = reference.name ? freightMap?.[reference.name] : undefined;
    result[warehouseIdentifier] = {
      reference,
      alias: fullFreight ? getAlias(fullFreight) : undefined
    };
  }

  return result;
}

export function getShortFreightLabel(name?: string, alias?: string): string {
  const shortID = (name || '').slice(0, 7);
  return alias ? `${alias} (${shortID})` : shortID;
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
