import { FreightCollection, FreightReference, TargetFreight } from '@ui/gen/api/v2/models';

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

// originKey is how a FreightCollection keys one origin's Freight, so that a
// Target's collection and its Stage's pair up origin by origin.
const originKey = (origin?: { kind?: string; name?: string }): string =>
  `${origin?.kind || 'Warehouse'}/${origin?.name || ''}`;

// entryCollection reads the Freight a status entry says a Target is on back
// into the shape a Stage's own Freight takes, so the two can be compared
// origin by origin. An entry names each piece of Freight and its origin and
// no more; resolve, when given, fills in the artifacts it carries, which is
// what a reader recognises it by. A Target with no Freight has no collection.
export const entryCollection = (
  freight: TargetFreight[] | undefined,
  resolve?: (name: string) => FreightReference | undefined
): FreightCollection | undefined => {
  if (!freight?.length) {
    return undefined;
  }
  return {
    items: Object.fromEntries(
      freight.map(({ name = '', origin }) => [
        originKey(origin),
        { ...resolve?.(name), name, origin }
      ])
    )
  };
};
