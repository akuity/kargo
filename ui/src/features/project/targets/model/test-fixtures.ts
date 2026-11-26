import {
  Stage,
  Target,
  TargetStageStatus,
  TargetStatusEntry,
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

// fleetStage builds a target-aware Stage, on the given Freight when one is
// named.
export const fleetStage = (name: string, selectors: V1LabelSelector[], freightName?: string) =>
  stage(name, {
    spec: { requestedFreight: [], targets: { selectors } },
    status: freightName
      ? {
          freightHistory: [
            {
              id: `collection-${freightName}`,
              items: {
                'Warehouse/w': { name: freightName, origin: { kind: 'Warehouse', name: 'w' } }
              }
            }
          ]
        }
      : undefined
  });

// entry builds the status the server reports for a Target under one Stage.
// freightName names what the Target is on, from one Warehouse.
export const entry = (
  targetName: string,
  stageName: string,
  state: TargetStatusEntry['state'],
  partial: Partial<TargetStatusEntry> & { freightName?: string } = {}
): TargetStatusEntry => {
  const { freightName, ...rest } = partial;
  return {
    target: targetName,
    stage: stageName,
    state,
    inFlight: state === 'Promoting' || state === 'Pending',
    behind: state === 'Behind',
    freight: freightName ? [{ name: freightName, origin: { kind: 'Warehouse', name: 'w' } }] : [],
    ...rest
  };
};

// health builds the entry a Target's own status carries for one Stage: its
// health, which is the one thing the statuses do not report.
export const health = (status: string): TargetStageStatus => ({ health: { status } });
