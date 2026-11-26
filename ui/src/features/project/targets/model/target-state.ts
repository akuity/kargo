import { GlobalToken } from 'antd';

import { TargetState as ServerTargetState } from '@ui/gen/api/v2/models';

// TargetState is where a Target stands with a Stage that governs it, as the
// server derives it from the Promotions and PromotionRollouts on record. It
// is the one word every surface uses for a Target, or for a Target under one
// of the Stages that govern it.
//
// It says how promotion to the Target went, not how the Target is doing:
// nothing records a Target's health, so none of these words speaks of it.
export type TargetState = ServerTargetState;

// StateColorToken names the Ant theme token a state is painted with. Resolve
// it against the live theme with theme.useToken() so the page follows dark
// mode, as the Promotion phase presentations do.
export type StateColorToken = Extract<
  keyof GlobalToken,
  | 'colorSuccess'
  | 'colorWarning'
  | 'colorInfo'
  | 'colorError'
  | 'colorTextQuaternary'
  | 'colorFillSecondary'
>;

export type TargetStatePresentation = {
  // label names the state in a legend, a tile heading or a table cell.
  label: string;
  // short names it after a count, as in "113 behind".
  short: string;
  colorToken: StateColorToken;
  // A hollow state draws as an outline rather than a fill wherever one Target
  // is one mark: nothing has promoted to it yet, so it reads as an empty slot.
  hollow?: boolean;
  // attention marks a state worth looking at: it sorts to the front of a grid
  // and names what is wrong in a group's caption.
  attention?: boolean;
};

// targetStatePresentations is the one mapping from the server's states to the
// words and colours the Targets surfaces paint with. Each reuses a mark the UI
// already had: a Promotion's phase colours for Promoting, Pending and Failed,
// the rollout page's "Not attempted" grey, the hollow "never promoted" slot,
// and the warning colour that has always meant behind the Stage.
export const targetStatePresentations: Record<TargetState, TargetStatePresentation> = {
  UpToDate: { label: 'Up to date', short: 'up to date', colorToken: 'colorSuccess' },
  Behind: { label: 'Behind Stage', short: 'behind', colorToken: 'colorWarning' },
  Promoting: { label: 'Promoting', short: 'promoting', colorToken: 'colorInfo' },
  Pending: { label: 'Pending', short: 'pending', colorToken: 'colorFillSecondary' },
  Failed: { label: 'Failed', short: 'failed', colorToken: 'colorError', attention: true },
  NotAttempted: {
    label: 'Not attempted',
    short: 'not attempted',
    colorToken: 'colorTextQuaternary'
  },
  NeverPromoted: {
    label: 'Never promoted',
    short: 'never promoted',
    colorToken: 'colorFillSecondary',
    hollow: true
  }
};

// stateOrder lists the states in the order every bar draws its segments and
// every legend and filter lists its counts: what is settled, what has
// drifted, what is moving, then what went wrong, what was left out, and what
// nothing has reached.
export const stateOrder: readonly TargetState[] = [
  'UpToDate',
  'Behind',
  'Promoting',
  'Pending',
  'Failed',
  'NotAttempted',
  'NeverPromoted'
];

// statePrecedence lists the states worst first, as the server counts them. A
// Target governed by several Stages shows the worst state any of them
// reports: a Target is only as far along as the Stage it is furthest behind
// for.
export const statePrecedence: readonly TargetState[] = [
  'Promoting',
  'Pending',
  'Failed',
  'NotAttempted',
  'NeverPromoted',
  'Behind',
  'UpToDate'
];

// isTargetState reports whether a word is one of the vocabulary.
export const isTargetState = (state?: string): state is TargetState =>
  !!state && Object.hasOwn(targetStatePresentations, state);

// toTargetState reads a state off the wire. A word this UI does not know --
// from a newer server, say -- is drawn as nothing having reached the Target,
// which claims the least.
export const toTargetState = (state?: string): TargetState =>
  isTargetState(state) ? state : 'NeverPromoted';

// worstState is the state a Target shows when several Stages govern it. A
// Target no Stage governs has never been promoted to by any of them.
export const worstState = (states: TargetState[]): TargetState =>
  statePrecedence.find((state) => states.includes(state)) ?? 'NeverPromoted';

export type StateCounts = Record<TargetState, number>;

// emptyStateCounts returns a fresh tally with every state at zero, so callers
// may index any state without a fallback.
export const emptyStateCounts = (): StateCounts => ({
  UpToDate: 0,
  Behind: 0,
  Promoting: 0,
  Pending: 0,
  Failed: 0,
  NotAttempted: 0,
  NeverPromoted: 0
});

// toStateCounts reads a tally off the wire, every state present whatever the
// server left out.
export const toStateCounts = (counts?: Partial<Record<string, number>>): StateCounts => {
  const tally = emptyStateCounts();
  for (const [state, count] of Object.entries(counts || {})) {
    if (isTargetState(state)) {
      tally[state] = count || 0;
    }
  }
  return tally;
};

// totalStateCount sums a tally.
export const totalStateCount = (counts: StateCounts): number =>
  stateOrder.reduce((sum, state) => sum + counts[state], 0);

// describeStates reads a tally back as a sentence, so a bar carries a label
// for readers who cannot see its colours.
export const describeStates = (counts: StateCounts): string => {
  const parts = stateOrder
    .filter((state) => counts[state])
    .map((state) => `${counts[state]} ${targetStatePresentations[state].short}`);
  return parts.length ? parts.join(', ') : 'no Targets';
};
