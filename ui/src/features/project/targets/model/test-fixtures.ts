import {
  PromotionRequest,
  Stage,
  TargetStageSummary,
  TargetSummary,
  V1LabelSelector
} from '@ui/gen/api/v2/models';

// Builders shared by the Target util suites, so each reads as the case it is
// about rather than as object literals.

// target builds a Target's summary. Its stages are the Stages that govern it,
// as the server settles them, each with what that Stage has done to it.
export const target = (
  name: string,
  labels?: Record<string, string>,
  stages?: Record<string, TargetStageSummary>
): TargetSummary => ({ name, labels, stages });

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

// forStage builds what a Target's summary says about one Stage that governs
// it: the collection that Stage last landed on it, its health since, and the
// phase of the Stage's latest Promotion to it. An undefined phase means no
// Promotion at all; an empty one, a Promotion not yet picked up.
export const forStage = (
  collectionID?: string,
  health?: string,
  freight?: string,
  phase?: string
): TargetStageSummary => ({
  collectionID,
  freight: freight ? [freight] : undefined,
  health: health ? { status: health } : undefined,
  lastPromotion: phase === undefined ? undefined : { name: 'promotion', phase }
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
