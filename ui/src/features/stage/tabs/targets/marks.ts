import {
  PromotionPhase,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { TargetRow } from '@ui/features/project/targets/model/target-rows';
import { TargetState } from '@ui/features/project/targets/model/target-state';

import { governing } from './summary';

// ColorMode is what the squares are painted for: how the latest round went for
// each Target, which Freight each is running, or how healthy each is. One
// fleet, read three ways.
export type ColorMode = 'promotion' | 'freight' | 'health';

export const colorModes: { value: ColorMode; label: string }[] = [
  { value: 'promotion', label: 'Last promotion' },
  { value: 'freight', label: 'Freight' },
  { value: 'health', label: 'Health' }
];

// Mark is one category of square: how it is painted, what the legend calls it,
// and whether it is something to look at. Attention categories sort to the
// front of a grid and name what is wrong in a group's caption.
export type Mark = {
  key: string;
  label: string;
  // short names the category after a count, as in "6 failed". Defaults to the
  // label.
  short?: string;
  color: string;
  hollow?: boolean;
  attention?: boolean;
  // order is where the category sits in the legend.
  order: number;
};

// Marking is one colour vocabulary: the categories it can paint, and which one
// a given Target falls into.
export type Marking = {
  legend: Mark[];
  mark: (row: TargetRow) => Mark;
};

// MarkColors are the live theme's colours, resolved once by the component and
// handed in, so these builders stay pure and testable.
export type MarkColors = {
  phase: Record<PromotionPhase, string>;
  state: Record<TargetState, string>;
  // freight is a categorical palette, cycled over the Freight the fleet runs.
  freight: string[];
};

const notInRound = 'NotInRound';
const neverPromotedKey = 'NeverPromoted';
const notAttempted = 'NotAttempted';
const noFreight = 'NoFreight';

// lookup indexes a legend by category key. Each marking supplies its own
// fallback for what the vocabulary does not name, since what to fall back to
// differs by vocabulary.
const lookup = (legend: Mark[]): Map<string, Mark> =>
  new Map(legend.map((mark) => [mark.key, mark]));

// promotionMarking paints each Target by how the Stage's latest round went for
// it, which for most fleets is the only reading there is: Targets are not
// health-checked yet.
export const promotionMarking = (colors: MarkColors, blocked = false): Marking => {
  // A blocked round never fanned out, so it did nothing to any Target -- not
  // even to the ones it names. One category for the whole fleet, matching
  // what the Last round tile says above it.
  if (blocked) {
    const mark: Mark = {
      key: notAttempted,
      label: 'Not attempted',
      short: 'not attempted',
      color: colors.state.Unknown,
      order: 0
    };
    return { legend: [mark], mark: () => mark };
  }

  const legend: Mark[] = [
    ...promotionPhases.map((phase, index) => ({
      key: phase,
      label: phase,
      short: phase.toLowerCase(),
      color: colors.phase[phase],
      attention: phase === 'Failed' || phase === 'Errored',
      order: index
    })),
    {
      key: notInRound,
      label: 'Not in round',
      short: 'not in round',
      color: colors.state.Unknown,
      order: promotionPhases.length
    },
    {
      key: neverPromotedKey,
      label: 'Never promoted',
      short: 'never promoted',
      color: colors.state.NotPromoted,
      hollow: true,
      order: promotionPhases.length + 1
    }
  ];
  const byKey = lookup(legend);
  return {
    legend,
    mark: (row) => {
      const phase = governing(row)?.phase;
      const named = phase && byKey.get(phase);
      if (named) {
        return named;
      }
      // The round says nothing about this Target. Freight on it means some
      // earlier round reached it; none means nothing ever has.
      return byKey.get(row.freight.length ? notInRound : neverPromotedKey) as Mark;
    }
  };
};

// healthStates maps each health status onto the state vocabulary the rest of
// the Targets pages paint with, so Unhealthy is the same red everywhere.
const healthStates: { status: string; state: TargetState; attention?: boolean }[] = [
  { status: 'Healthy', state: 'Healthy' },
  { status: 'Progressing', state: 'Progressing' },
  { status: 'Degraded', state: 'Unhealthy', attention: true },
  { status: 'Unhealthy', state: 'Unhealthy', attention: true },
  { status: 'Unknown', state: 'Unknown' }
];

// healthMarking paints each Target by its health with respect to this Stage's
// Freight. A Target nothing has assessed is Unknown, as its own status would
// have it.
export const healthMarking = (colors: MarkColors): Marking => {
  const legend: Mark[] = healthStates.map(({ status, state, attention }, index) => ({
    key: status,
    label: status,
    short: status.toLowerCase(),
    color: colors.state[state],
    attention,
    order: index
  }));
  const byKey = lookup(legend);
  const unknown = byKey.get('Unknown') as Mark;
  return {
    legend,
    mark: (row) => byKey.get(governing(row)?.health?.status || '') || unknown
  };
};

// freightKey names the Freight a Target runs. A Stage may request Freight from
// several origins, so a Target runs a collection of it rather than one piece,
// and the whole collection is the category.
const freightKey = (names: string[]): string => names.join(' + ') || noFreight;

// freightMarking paints each Target by what it is running. The Stage's own
// current Freight leads the legend -- it is what every Target is on its way to
// -- and everything else is a Target left behind, which is what makes the
// other colours worth looking at.
export const freightMarking = (
  rows: TargetRow[],
  stageFreight: string[],
  colors: MarkColors,
  freightLabel: (name: string) => string
): Marking => {
  const counts = new Map<string, { names: string[]; count: number }>();
  for (const row of rows) {
    const key = freightKey(row.freight);
    const entry = counts.get(key) || { names: row.freight, count: 0 };
    entry.count++;
    counts.set(key, entry);
  }

  const current = freightKey(stageFreight);
  // The Stage's own Freight first, then whatever most of the fleet is on, so
  // the colours read in the order a fleet rolls out.
  const ranked = [...counts.entries()]
    .filter(([key]) => key !== noFreight && key !== current)
    .sort(([lhsKey, lhs], [rhsKey, rhs]) => rhs.count - lhs.count || lhsKey.localeCompare(rhsKey));

  const named = [
    ...(stageFreight.length ? [[current, { names: stageFreight, count: 0 }] as const] : []),
    ...ranked
  ];
  const legend: Mark[] = named.map(([key, { names }], index) => ({
    key,
    label: names.map(freightLabel).join(' + '),
    short: `on ${names.map(freightLabel).join(' + ')}`,
    color: colors.freight[index % colors.freight.length],
    attention: key !== current,
    order: index
  }));
  legend.push({
    key: noFreight,
    label: 'none',
    short: 'without Freight',
    color: colors.state.NotPromoted,
    hollow: true,
    order: legend.length
  });

  const byKey = lookup(legend);
  const none = byKey.get(noFreight) as Mark;
  return { legend, mark: (row) => byKey.get(freightKey(row.freight)) || none };
};

export const marking = (
  mode: ColorMode,
  rows: TargetRow[],
  stageFreight: string[],
  colors: MarkColors,
  freightLabel: (name: string) => string,
  // Whether the Stage's latest round is blocked, which only the promotion
  // vocabulary has anything to say about.
  blocked = false
): Marking => {
  switch (mode) {
    case 'freight':
      return freightMarking(rows, stageFreight, colors, freightLabel);
    case 'health':
      return healthMarking(colors);
    default:
      return promotionMarking(colors, blocked);
  }
};

export type MarkTally = { mark: Mark; count: number };

// tally counts the rows in each category, in legend order, leaving out the
// categories nothing is in: a reader should not have to skip past zeroes.
export const tally = (rows: TargetRow[], marks: Marking): MarkTally[] => {
  const counts = new Map<string, number>();
  for (const row of rows) {
    const { key } = marks.mark(row);
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return marks.legend
    .filter((mark) => counts.get(mark.key))
    .map((mark) => ({ mark, count: counts.get(mark.key) as number }));
};

// attentionTally is what a group's caption says under its name: the largest
// category worth looking at, or nothing when the whole group is fine.
export const attentionTally = (rows: TargetRow[], marks: Marking): MarkTally | undefined =>
  tally(rows, marks)
    .filter(({ mark }) => mark.attention)
    .sort((lhs, rhs) => rhs.count - lhs.count)[0];

// byMark orders Targets the way a grid draws them: whatever needs attention
// first, then by the legend, then by name, so a failure is always in the
// top-left corner of its group.
export const byMark =
  (marks: Marking) =>
  (lhs: TargetRow, rhs: TargetRow): number => {
    const left = marks.mark(lhs);
    const right = marks.mark(rhs);
    if (!!left.attention !== !!right.attention) {
      return left.attention ? -1 : 1;
    }
    return left.order - right.order || (lhs.target.name || '').localeCompare(rhs.target.name || '');
  };
