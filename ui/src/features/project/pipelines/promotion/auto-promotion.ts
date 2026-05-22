import type { Freight, FreightReference, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import type { AutoPromotionCandidate } from '@ui/gen/api/v2/models/autoPromotionCandidate';

type OriginLike = {
  kind?: string;
  name?: string;
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
