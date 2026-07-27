import type { AutoPromotionHold, Stage } from '@ui/gen/api/v2/models';

type OriginLike = {
  kind?: string;
  name?: string;
};

export type AutoPromotionHoldEntry = {
  key: string;
  hold: AutoPromotionHold;
  origin?: OriginLike;
};

export const originKey = (origin?: OriginLike) => {
  if (!origin?.kind || !origin?.name) {
    return '';
  }
  return `${origin.kind}/${origin.name}`;
};

export const originLabel = (origin?: OriginLike) => {
  if (!origin?.kind || !origin?.name) {
    return 'this origin';
  }
  return `${origin.kind}/${origin.name}`;
};

export const getAutoPromotionHold = (stage: Stage | undefined, origin?: OriginLike) => {
  const key = originKey(origin);
  if (!key) {
    return undefined;
  }
  return stage?.status?.autoPromotionHolds?.[key];
};

export const stageHasAutoPromotionHold = (stage: Stage | undefined): boolean =>
  Object.keys(stage?.status?.autoPromotionHolds || {}).length > 0;

export const getAutoPromotionHoldEntries = (stage: Stage | undefined): AutoPromotionHoldEntry[] => {
  const holds = (stage?.status?.autoPromotionHolds ?? {}) as Record<string, AutoPromotionHold>;
  return Object.entries(holds)
    .map(([key, hold]) => ({ key, hold, origin: hold.origin }))
    .sort((lhs, rhs) => lhs.key.localeCompare(rhs.key));
};

export const holdStateMessage = (stage?: Stage, origin?: OriginLike) => {
  if (origin) {
    return `Auto-promotion paused: ${originLabel(origin)}`;
  }
  const origins = getAutoPromotionHoldEntries(stage).map((entry) => entry.key);
  if (origins.length === 0) {
    return 'Auto-promotion paused.';
  }
  return `Auto-promotion paused: ${origins.join(', ')}`;
};
