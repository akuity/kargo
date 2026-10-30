import { isPromotionRequestPhaseTerminal } from '@ui/features/stage/utils/promotion-request';
import { FreightCollection, FreightReference, PromotionRequest } from '@ui/gen/api/v2/models';

// roundPhase is the phase a round reports for one Target: the child Promotion's
// when there is one, the round's own when it ended before a child could be
// created, and Pending while the round has yet to reach the Target. A Target
// the round never named has no phase -- the round says nothing about it.
export const roundPhase = (
  request: PromotionRequest | undefined,
  target: string | undefined
): string | undefined => {
  if (!(request?.spec?.targets || []).some((named) => named.name === target)) {
    return undefined;
  }
  const status = request?.status?.targets?.find((named) => named.name === target);
  if (status) {
    // A child that exists but has not been picked up yet has no phase; it is
    // Pending in every sense that matters here.
    return status.phase || 'Pending';
  }
  const requestPhase = request?.status?.phase;
  return isPromotionRequestPhaseTerminal(requestPhase) ? requestPhase : 'Pending';
};

// collectionUpToDate compares what a Target runs for a Stage against what the
// Stage runs. Collection IDs are deterministic in their contents, so equal IDs
// mean equal Freight; either side missing leaves the question open.
export const collectionUpToDate = (
  currentFreight: FreightCollection | undefined,
  collectionID: string | undefined
): boolean | undefined =>
  currentFreight?.id && collectionID ? currentFreight.id === collectionID : undefined;

// freightNames lists the Freight a collection holds, one per origin, in a
// stable order.
export const freightNames = (collection?: FreightCollection): string[] =>
  Object.values(collection?.items || {})
    .map((reference) => reference.name || '')
    .filter(Boolean)
    .sort();

// OriginFreight is one origin of a Target's Freight, side by side with the
// same origin of its Stage's.
export type OriginFreight = {
  // The Warehouse the Freight comes from.
  origin: string;
  // What the Target runs from this origin, absent when it has none from it.
  current?: FreightReference;
  // What the Stage is on from this origin.
  stage?: FreightReference;
  // Whether the Stage has moved this origin on without the Target.
  behind: boolean;
};

// freightByOrigin pairs what a Target runs with what its Stage runs, origin by
// origin, sorted by origin. A Stage requesting Freight from many Warehouses --
// a cluster's worth of addons, say -- is behind on some and not on others, and
// the collection as a whole only says that something differs. A Target with no
// Freight yet is behind on every origin the Stage has.
export const freightByOrigin = (
  current: FreightCollection | undefined,
  stage: FreightCollection | undefined
): OriginFreight[] => {
  const keys = new Set([...Object.keys(current?.items || {}), ...Object.keys(stage?.items || {})]);
  return [...keys]
    .map((key) => {
      const currentRef = current?.items?.[key];
      const stageRef = stage?.items?.[key];
      return {
        origin: currentRef?.origin?.name || stageRef?.origin?.name || key.split('/').pop() || key,
        current: currentRef,
        stage: stageRef,
        behind: !!stageRef?.name && stageRef.name !== currentRef?.name
      };
    })
    .sort((lhs, rhs) => lhs.origin.localeCompare(rhs.origin));
};
