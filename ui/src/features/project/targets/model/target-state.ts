import { GlobalToken } from 'antd';

// TargetState is the one word a surface uses for a Target, or for a Target
// under one of the Stages that govern it. Behind is a state of its own rather
// than a flavour of Healthy: a Target running Freight its Stage has moved on
// from is working, but it is not where the pipeline wants it.
export type TargetState =
  'Healthy' | 'Behind' | 'Progressing' | 'Unhealthy' | 'Unknown' | 'NotPromoted';

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
  // label names the state in a legend or a tile heading.
  label: string;
  // short names it after a count, as in "113 behind".
  short: string;
  colorToken: StateColorToken;
  // A hollow state draws as an outline rather than a fill wherever one Target
  // is one mark: nothing has promoted to it yet, so it reads as an empty slot.
  hollow?: boolean;
};

export const targetStatePresentations: Record<TargetState, TargetStatePresentation> = {
  Healthy: { label: 'Healthy', short: 'healthy', colorToken: 'colorSuccess' },
  Behind: { label: 'Behind Stage', short: 'behind', colorToken: 'colorWarning' },
  Progressing: { label: 'Progressing', short: 'progressing', colorToken: 'colorInfo' },
  Unhealthy: { label: 'Unhealthy', short: 'unhealthy', colorToken: 'colorError' },
  Unknown: { label: 'Unknown', short: 'unknown', colorToken: 'colorTextQuaternary' },
  NotPromoted: {
    label: 'Not promoted',
    short: 'not promoted',
    colorToken: 'colorFillSecondary',
    hollow: true
  }
};

// stateOrder lists the states in the order every bar draws its segments and
// every legend lists its counts: what is settled, what has drifted, what is
// moving, then what is wrong, unknown, or absent.
export const stateOrder: readonly TargetState[] = [
  'Healthy',
  'Behind',
  'Progressing',
  'Unhealthy',
  'Unknown',
  'NotPromoted'
];

// statePrecedence lists the states worst first. A Target governed by several
// Stages shows the worst state any of them reports: a Target is only as well
// off as its worst Stage. A round still moving outranks a health reading,
// which describes the Freight that round is on its way to replace.
export const statePrecedence: readonly TargetState[] = [
  'Progressing',
  'Unhealthy',
  'NotPromoted',
  'Unknown',
  'Behind',
  'Healthy'
];

export type StateCounts = Record<TargetState, number>;

// emptyStateCounts returns a fresh tally with every state at zero, so callers
// may index any state without a fallback.
export const emptyStateCounts = (): StateCounts => ({
  Healthy: 0,
  Behind: 0,
  Progressing: 0,
  Unhealthy: 0,
  Unknown: 0,
  NotPromoted: 0
});

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
