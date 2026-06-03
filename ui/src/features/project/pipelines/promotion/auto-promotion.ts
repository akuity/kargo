import type {
  AutoPromotionHold,
  Freight,
  FreightReference,
  Stage
} from '@ui/gen/api/v1alpha1/generated_pb';
import type { AutoPromotionCandidate } from '@ui/gen/api/v2/models/autoPromotionCandidate';

export type OriginLike = {
  kind?: string;
  name?: string;
};

export type AutoPromotionHoldEntry = {
  key: string;
  hold: AutoPromotionHold;
  origin?: OriginLike;
  focused: boolean;
};

export const autoPromotionHoldStateActive = 'Active';
export const autoPromotionHoldStatePending = 'Pending';

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

export const getAutoPromotionCandidate = (
  candidates: AutoPromotionCandidate[] | undefined,
  origin?: OriginLike
) => {
  const key = originKey(origin);
  if (!key) {
    return undefined;
  }
  return candidates?.find((candidate) => originKey(candidate.origin) === key);
};

export const getAutoPromotionCandidateName = (
  candidates: AutoPromotionCandidate[] | undefined,
  freight: Pick<Freight | FreightReference, 'origin'> | undefined
) => getAutoPromotionCandidate(candidates, freight?.origin)?.freight?.name;

export const getAutoPromotionHold = (stage: Stage | undefined, origin?: OriginLike) => {
  const key = originKey(origin);
  if (!key) {
    return undefined;
  }
  return stage?.status?.autoPromotionHolds?.[key];
};

export const originFromKey = (key: string): OriginLike | undefined => {
  const [kind, name] = key.split('/');
  if (!kind || !name) {
    return undefined;
  }
  return { kind, name };
};

export const getAutoPromotionHoldEntries = (
  stage: Stage | undefined,
  focusOrigin?: OriginLike
): AutoPromotionHoldEntry[] => {
  const focusOriginKey = originKey(focusOrigin);
  const holds = stage?.status?.autoPromotionHolds || {};

  return Object.entries(holds)
    .map(([key, hold]) => ({
      key,
      hold,
      origin: hold?.freight?.origin || originFromKey(key),
      focused: Boolean(focusOriginKey && focusOriginKey === key)
    }))
    .sort((lhs, rhs) => {
      if (lhs.focused !== rhs.focused) {
        return lhs.focused ? -1 : 1;
      }
      if (lhs.hold.state !== rhs.hold.state) {
        return lhs.hold.state === autoPromotionHoldStateActive ? -1 : 1;
      }
      return lhs.key.localeCompare(rhs.key);
    });
};
