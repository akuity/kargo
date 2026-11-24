import {
  PromotionRequest,
  Stage,
  Target,
  TargetStageStatus,
  V1LabelSelector
} from '@ui/gen/api/v2/models';

// Builders shared by the Target util suites, so each reads as the case it is
// about rather than as object literals.
export const target = (
  name: string,
  labels?: Record<string, string>,
  partial: Partial<Target> = {}
): Target => ({
  metadata: { name, labels },
  ...partial
});

export const stage = (name: string, partial: Partial<Stage> = {}): Stage => ({
  metadata: { name },
  // Required by the type. An empty spec is still a classic Stage: what makes
  // one target-aware is the targets block, which this leaves out.
  spec: { requestedFreight: [] },
  ...partial
});

export const fleetStage = (name: string, selectors: V1LabelSelector[], collectionID?: string) =>
  stage(name, {
    spec: { requestedFreight: [], targets: { selectors } },
    status: collectionID ? { freightHistory: [{ id: collectionID }] } : undefined
  });

// forStage builds the entry a Target's status carries for one Stage: the
// Freight collection that Stage last landed on it, and its health since.
export const forStage = (
  collectionID?: string,
  health?: string,
  freight?: string
): TargetStageStatus => ({
  currentFreight: collectionID
    ? { id: collectionID, items: freight ? { 'Warehouse/w': { name: freight } } : {} }
    : undefined,
  health: health ? { status: health } : undefined
});

export const round = (
  stageName: string,
  targets: string[],
  partial: Partial<PromotionRequest['status']> = {}
): PromotionRequest => ({
  metadata: { name: `${stageName}.01` },
  spec: { stage: stageName, freight: 'freight-new', targets: targets.map((name) => ({ name })) },
  status: partial
});
