import {
  FreightCollection,
  FreightReference,
  Health,
  Stage,
  Target,
  TargetStatusEntry
} from '@ui/gen/api/v2/models';

import { entryCollection, freightNames } from './target-freight';
import { byName } from './target-stages';
import {
  StateCounts,
  TargetState,
  emptyStateCounts,
  statePrecedence,
  toTargetState,
  worstState
} from './target-state';

// GoverningStage is one Stage that governs a Target, with where the Target
// stands with it. All of it is scoped to the Stage: a Target governed by
// several may be up to date for one and behind for another.
export type GoverningStage = {
  stage: Stage;
  // The server's reading of the Target under this Stage, derived from the
  // whole history of Promotions and PromotionRollouts on record.
  entry: TargetStatusEntry;
  state: TargetState;
  // Whether the Freight the Target is on is not the Stage's current Freight.
  // It cuts across state: a Target that failed or was not attempted is
  // usually behind as well.
  behind: boolean;
  // What the Target is on for this Stage, absent until a Promotion from the
  // Stage to it has succeeded.
  currentFreight?: FreightCollection;
  // The Target's health with respect to this Stage's Freight, from the
  // Target's own status. Nothing writes it yet, so it is commonly absent; the
  // surfaces that show it do so only when something has.
  health?: Health;
};

export type TargetRow = {
  target: Target;
  stages: GoverningStage[];
  // The Target's state across every Stage that governs it: the worst of them.
  state: TargetState;
  // Whether any governing Stage has the Target on older Freight than its own.
  behind: boolean;
  // The Freight the Target is on, by name, across its governing Stages.
  freight: string[];
  // The one governing Stage this row has been read for, when the page is
  // grouped by Stage. state, behind and freight are then that Stage's reading
  // alone; stages stays whole either way, so a row always knows every Stage
  // that governs its Target. Absent on an unscoped row.
  scopedTo?: GoverningStage;
};

// targetRows pairs every Target with the status entries of the Stages that
// govern it, both sorted by name.
//
// The entries say which Stages govern a Target and where it stands with each;
// the Targets themselves say what else there is to show of one -- its params,
// when it was made, and any health recorded for it -- and include those no
// Stage governs, which have no entry. Either list may run a poll ahead of the
// other, so a Target with entries and no object yet is drawn from its entries'
// labels, and an entry for a Stage not loaded yet stands in a Stage of that
// name.
export const targetRows = (
  targets: Target[],
  entries: TargetStatusEntry[],
  stages: Stage[],
  // resolve fills in what a piece of Freight carries, when known.
  resolve?: (name: string) => FreightReference | undefined
): TargetRow[] => {
  const stagesByName = new Map(stages.map((stage) => [stage.metadata?.name || '', stage]));
  const entriesByTarget = new Map<string, TargetStatusEntry[]>();
  for (const entry of entries) {
    const name = entry.target || '';
    entriesByTarget.set(name, [...(entriesByTarget.get(name) || []), entry]);
  }

  const byTargetName = new Map(targets.map((target) => [target.metadata?.name || '', target]));
  for (const [name, mine] of entriesByTarget) {
    if (!byTargetName.has(name)) {
      byTargetName.set(name, { metadata: { name, labels: mine[0]?.labels } });
    }
  }

  return [...byTargetName.values()].sort(byName).map((target) => {
    const name = target.metadata?.name || '';
    const governed = (entriesByTarget.get(name) || [])
      .map<GoverningStage>((entry) => {
        const stageName = entry.stage || '';
        return {
          stage: stagesByName.get(stageName) || {
            metadata: { name: stageName },
            spec: { requestedFreight: [] }
          },
          entry,
          state: toTargetState(entry.state),
          behind: !!entry.behind,
          currentFreight: entryCollection(entry.freight, resolve),
          health: target.status?.stages?.[stageName]?.health
        };
      })
      .sort((lhs, rhs) => byName(lhs.stage, rhs.stage));
    return {
      target,
      stages: governed,
      state: worstState(governed.map(({ state }) => state)),
      behind: governed.some(({ behind }) => behind),
      freight: [...new Set(governed.flatMap(({ currentFreight }) => freightNames(currentFreight)))]
    };
  });
};

// scopeToStage reads a row as one of its governing Stages sees it. Grouped by
// Stage, a Target governed by several appears under each, and under each it
// should show where it stands for that Stage alone -- not the worst of all of
// them, which would paint a Stage's group with trouble from another Stage.
//
// stages is deliberately left whole. The reading narrows to one Stage; the
// facts about the Target do not, and a reader who opens the row from a Stage's
// group still needs to see every Stage that governs it -- including the one
// the trouble is actually on.
export const scopeToStage = (row: TargetRow, governing: GoverningStage): TargetRow => ({
  ...row,
  state: governing.state,
  behind: governing.behind,
  freight: freightNames(governing.currentFreight),
  scopedTo: governing
});

// aheadFreight names the Freight that Stages which have left this Target behind
// have moved on to.
//
// A row read for one Stage answers for that Stage alone: a Target up to date
// for the Stage whose group it sits in is not behind there, whatever another
// Stage has since promoted.
export const aheadFreight = (row: TargetRow): string[] => [
  ...new Set(
    (row.scopedTo ? [row.scopedTo] : row.stages)
      .filter(({ behind }) => behind)
      .flatMap(({ stage }) => freightNames(stage.status?.freightHistory?.[0]))
  )
];

// stateCounts tallies rows by the state they show. Every state is present, so
// callers may index any of them without a fallback.
export const stateCounts = (rows: TargetRow[]): StateCounts =>
  rows.reduce((counts, row) => {
    counts[row.state]++;
    return counts;
  }, emptyStateCounts());

// behindCount counts the Targets any governing Stage has left on older Freight.
// It overlaps the state tally on purpose: a Target that is behind may also have
// failed, be mid-promotion, or have been left out of a rollout.
export const behindCount = (rows: TargetRow[]): number => rows.filter((row) => row.behind).length;

// inFlight reports whether a promotion is on its way to any of the rows'
// Targets, which is what the status list is polled for.
export const inFlight = (entries: TargetStatusEntry[]): boolean =>
  entries.some((entry) => entry.inFlight);

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

// byWorstState orders rows worst first, so whatever needs attention is at the
// top of a long list and survives into a short preview of it.
export const byWorstState = (rows: TargetRow[]): TargetRow[] =>
  [...rows].sort(
    (lhs, rhs) =>
      statePrecedence.indexOf(lhs.state) - statePrecedence.indexOf(rhs.state) ||
      (lhs.target.metadata?.name || '').localeCompare(rhs.target.metadata?.name || '')
  );
