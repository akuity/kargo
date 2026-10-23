import { HealthStatus, healthStatusToEnum } from '@ui/features/common/health-status/utils';
import { roundBlock } from '@ui/features/stage/utils/promotion-rollout';
import { FreightCollection, Health, PromotionRollout, Stage, Target } from '@ui/gen/api/v2/models';

import { collectionUpToDate, freightNames, roundPhase } from './round-utils';
import { byName, pipelineOrder, stageGovernsTarget, targetAwareStages } from './target-stages';
import { StateCounts, TargetState, emptyStateCounts, statePrecedence } from './target-state';

// Rounds is the PromotionRollout each target-aware Stage is running now, keyed
// by Stage name. A Stage with no round in flight simply has no entry.
export type Rounds = Record<string, PromotionRollout | undefined>;

// GoverningStage is one Stage that governs a Target, with everything that
// Stage observes about it. All of it is scoped to the Stage: a Target governed
// by several may be up to date and healthy for one while behind for another.
export type GoverningStage = {
  stage: Stage;
  // The Target's health with respect to this Stage's Freight, absent until the
  // first assessment.
  health?: Health;
  // What the Target runs for this Stage, absent until a Promotion from the
  // Stage to this Target has succeeded.
  currentFreight?: FreightCollection;
  // Whether the Target has what the Stage has. False when it has fallen
  // behind; undefined when there is nothing to tell either way.
  upToDate?: boolean;
  // The phase the Stage's current round reports for this Target, when it names
  // it at all.
  phase?: string;
  state: TargetState;
};

// isUpToDate reports whether a Target has what its Stage has. With no recorded
// collection to compare, the Stage's latest round stands in: it promotes the
// Stage's own Freight, so a Target it succeeded on has caught up.
const isUpToDate = (
  currentFreight: FreightCollection | undefined,
  collectionID: string | undefined,
  phase: string | undefined
): boolean | undefined => {
  const compared = collectionUpToDate(currentFreight, collectionID);
  if (compared !== undefined) {
    return compared;
  }
  switch (phase) {
    case 'Succeeded':
      return true;
    case 'Failed':
    case 'Errored':
      return false;
    default:
      return undefined;
  }
};

// settledState is where a Target stands when nothing is happening to it: behind
// when its Stage has moved on without it, and otherwise where it should be.
const settledState = (governed: Omit<GoverningStage, 'state'>): TargetState =>
  governed.upToDate === false ? 'Behind' : 'Healthy';

// governingStageState is the one word for a Target under one Stage.
const governingStageState = (governed: Omit<GoverningStage, 'state'>): TargetState => {
  // A round still moving says more than a health reading, which describes the
  // Freight that round is on its way to replace.
  if (governed.phase === 'Pending' || governed.phase === 'Running') {
    return 'Progressing';
  }

  switch (healthStatusToEnum(governed.health?.status)) {
    case HealthStatus.UNHEALTHY:
    case HealthStatus.DEGRADED:
      return 'Unhealthy';
    case HealthStatus.PROGRESSING:
      return 'Progressing';
    case HealthStatus.HEALTHY:
      return settledState(governed);
    case HealthStatus.UNKNOWN:
      return 'Unknown';
    default:
      break;
  }

  // No health recorded -- the ordinary case, since Targets are not
  // health-checked yet. Fall back to how the Stage's latest round went, as the
  // back end's own Target stats do until health is written.
  switch (governed.phase) {
    case 'Failed':
    case 'Errored':
      return 'Unhealthy';
    case 'Succeeded':
      return settledState(governed);
    default:
      break;
  }

  // The round says nothing about this Target. Freight on it means some earlier
  // promotion succeeded; none means nothing ever has.
  return governed.currentFreight ? settledState(governed) : 'NotPromoted';
};

// worstState is the state a Target shows when several Stages govern it.
export const worstState = (states: TargetState[]): TargetState => {
  for (const state of statePrecedence) {
    if (states.includes(state)) {
      return state;
    }
  }
  return 'NotPromoted';
};

export type TargetRow = {
  target: Target;
  stages: GoverningStage[];
  // The Target's state across every Stage that governs it.
  state: TargetState;
  // Whether any governing Stage has the Target on older Freight than its own
  // current. Tracked apart from state because a Target can be both behind and
  // healthy, and the summary tiles count the two separately.
  behind: boolean;
  // The Freight the Target runs, by name, across its governing Stages.
  freight: string[];
};

// targetRows pairs every Target with the Stages that govern it, both sorted by
// name, and reads each pairing's state from the Target's own status and from
// the round that Stage is running. A Target no Stage governs still gets a row,
// with no Stages.
export const targetRows = (
  targets: Target[],
  stages: Stage[],
  rounds: Rounds = {}
): TargetRow[] => {
  const governing = targetAwareStages(stages);
  // A round that never fanned out reached no Target. Settled once per Stage,
  // since the answer does not vary by Target.
  const reaching = new Map(
    governing.map((stage) => {
      const round = rounds[stage.metadata?.name || ''];
      return [stage.metadata?.name || '', round && !roundBlock(round) ? round : undefined];
    })
  );
  return [...targets].sort(byName).map((target) => {
    const name = target.metadata?.name || '';
    const governed = governing
      .filter((stage) => stageGovernsTarget(stage, target))
      .map<GoverningStage>((stage) => {
        const stageName = stage.metadata?.name || '';
        const forStage = target.status?.stages?.[stageName];
        const currentFreight = forStage?.currentFreight;
        const collectionID = stage.status?.freightHistory?.[0]?.id;
        const phase = roundPhase(reaching.get(stageName), name);
        const pairing = {
          stage,
          health: forStage?.health,
          currentFreight,
          upToDate: isUpToDate(currentFreight, collectionID, phase),
          phase
        };
        return { ...pairing, state: governingStageState(pairing) };
      });
    return {
      target,
      stages: governed,
      state: worstState(governed.map(({ state }) => state)),
      behind: governed.some(({ upToDate }) => upToDate === false),
      freight: [...new Set(governed.flatMap(({ currentFreight }) => freightNames(currentFreight)))]
    };
  });
};

// stateCounts tallies rows by the state they show. Every state is present, so
// callers may index any of them without a fallback.
export const stateCounts = (rows: TargetRow[]): StateCounts =>
  rows.reduce((counts, row) => {
    counts[row.state]++;
    return counts;
  }, emptyStateCounts());

// behindCount counts the Targets any governing Stage has left on older Freight.
// It overlaps the state tally on purpose: a Target that is behind may also be
// healthy, unhealthy, or mid-promotion, and the summary reports both readings.
export const behindCount = (rows: TargetRow[]): number => rows.filter((row) => row.behind).length;

export type FreightTally = { name: string; count: number };

// freightTally counts the Targets running each piece of Freight, commonest
// first. A Target governed by several Stages may run more than one, and counts
// once under each.
export const freightTally = (rows: TargetRow[]): FreightTally[] => {
  const counts = new Map<string, number>();
  for (const row of rows) {
    for (const name of row.freight) {
      counts.set(name, (counts.get(name) || 0) + 1);
    }
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((lhs, rhs) => rhs.count - lhs.count || lhs.name.localeCompare(rhs.name));
};

// StageRollout is how far one Stage has got in reaching the Targets it governs.
export type StageRollout = {
  stage: Stage;
  // The Freight the Stage itself is on, by name: what every Target it governs
  // is on its way to.
  freight: string[];
  governed: number;
  // Targets already on the Stage's current Freight collection.
  onLatest: number;
  // Targets this Stage is promoting to right now.
  inFlight: number;
  unhealthy: number;
  counts: StateCounts;
};

// stageRollouts summarizes each target-aware Stage over the rows given, so a
// search narrows the Stage cards with everything else. Stages come out in
// pipeline order, which is why this takes the Project's Stages rather than only
// the target-aware ones it summarizes.
export const stageRollouts = (rows: TargetRow[], stages: Stage[]): StageRollout[] => {
  const ordered = pipelineOrder(targetAwareStages(stages), stages);
  // One pass, tallying into each Stage's accumulator: this recomputes on every
  // keystroke in the search box.
  const tallies = new Map(
    ordered.map((stage) => [
      stage.metadata?.name || '',
      { governed: 0, onLatest: 0, counts: emptyStateCounts() }
    ])
  );
  for (const row of rows) {
    for (const pairing of row.stages) {
      const tally = tallies.get(pairing.stage.metadata?.name || '');
      if (!tally) {
        continue;
      }
      tally.governed++;
      tally.counts[pairing.state]++;
      if (pairing.upToDate === true) {
        tally.onLatest++;
      }
    }
  }
  return ordered.map((stage) => {
    const tally = tallies.get(stage.metadata?.name || '');
    const counts = tally?.counts ?? emptyStateCounts();
    return {
      stage,
      freight: freightNames(stage.status?.freightHistory?.[0]),
      governed: tally?.governed ?? 0,
      onLatest: tally?.onLatest ?? 0,
      inFlight: counts.Progressing,
      unhealthy: counts.Unhealthy,
      counts
    };
  });
};

// byWorstState orders rows worst first, so whatever needs attention is at the
// top of a long list and survives into a short preview of it.
export const byWorstState = (rows: TargetRow[]): TargetRow[] =>
  [...rows].sort(
    (lhs, rhs) =>
      statePrecedence.indexOf(lhs.state) - statePrecedence.indexOf(rhs.state) ||
      (lhs.target.metadata?.name || '').localeCompare(rhs.target.metadata?.name || '')
  );
