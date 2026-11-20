import {
  IconDefinition,
  faCancel,
  faCircleCheck,
  faCircleExclamation,
  faCircleMinus,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { GlobalToken, theme } from 'antd';

// PromotionPhase mirrors PromotionPhase on the back end. A PromotionRequest's
// summary counts child Promotions by these same phases, so both the Promotion
// and PromotionRequest UIs present them with one shared vocabulary.
export type PromotionPhase = 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Errored' | 'Aborted';

// PhaseColorToken names the Ant theme token a phase is painted with when it
// needs a real color rather than a Tag preset -- a progress bar segment, say.
// Resolve it against the live theme with theme.useToken() so it follows dark
// mode, unlike iconColor, which is fixed to the default seed.
export type PhaseColorToken = Extract<
  keyof GlobalToken,
  'colorSuccess' | 'colorError' | 'colorInfo' | 'colorFillSecondary' | 'colorTextQuaternary'
>;

export type PromotionPhasePresentation = {
  icon: IconDefinition;
  // iconColor is applied to a standalone FontAwesome icon.
  iconColor: string;
  // tagColor is an Ant Design Tag preset.
  tagColor: 'default' | 'processing' | 'success' | 'error';
  // colorToken paints the phase where a preset will not do. Succeeded is
  // green, Failed and Errored red, Running blue. Pending is a fill one step
  // darker than an empty track, so "not started" reads as outline rather than
  // status, and Aborted is a darker neutral so a deliberate cancellation is
  // neither a failure nor something still waiting.
  colorToken: PhaseColorToken;
  spin: boolean;
};

const neutral = { iconColor: 'aaa', tagColor: 'default', spin: false } as const;

const failed: PromotionPhasePresentation = {
  icon: faCircleExclamation,
  iconColor: theme.defaultSeed.colorError,
  tagColor: 'error',
  colorToken: 'colorError',
  spin: false
};

export const promotionPhasePresentations: Record<PromotionPhase, PromotionPhasePresentation> = {
  Pending: { ...neutral, icon: faHourglassStart, colorToken: 'colorFillSecondary' },
  Running: {
    ...neutral,
    icon: faCircleNotch,
    tagColor: 'processing',
    colorToken: 'colorInfo',
    spin: true
  },
  Succeeded: {
    icon: faCircleCheck,
    iconColor: theme.defaultSeed.colorSuccess,
    tagColor: 'success',
    colorToken: 'colorSuccess',
    spin: false
  },
  Failed: failed,
  Errored: failed,
  Aborted: { ...neutral, icon: faCancel, colorToken: 'colorTextQuaternary' }
};

// promotionPhases lists every phase in the order a summary should present
// them: what went well, what is still happening, then what did not.
export const promotionPhases: readonly PromotionPhase[] = [
  'Succeeded',
  'Running',
  'Pending',
  'Failed',
  'Errored',
  'Aborted'
];

// PhaseFilter is one chip in a row of phase filters. The first names no phase:
// it is "All", and selecting it clears the filter.
export type PhaseFilter<P extends string = PromotionPhase> = { phase?: P; count: number };

export const isPromotionPhase = (phase?: string): phase is PromotionPhase =>
  !!phase && Object.hasOwn(promotionPhasePresentations, phase);

// getPromotionPhasePresentation treats an unset or unknown phase as Pending,
// which is what a Promotion is until the controller says otherwise.
export const getPromotionPhasePresentation = (phase?: string): PromotionPhasePresentation =>
  promotionPhasePresentations[isPromotionPhase(phase) ? phase : 'Pending'];

// notAttempted is the one thing a round of fan-out says of a Target that no
// Promotion says of itself: the round named the Target and ended without ever
// promoting to it. A round that exceeds the failures it tolerates, or is
// aborted, stops fanning out, and the Targets it had yet to reach get no
// Promotion at all. Nothing is pending for them and nothing failed on them.
export const notAttempted = 'NotAttempted';

// RoundTargetPhase is what a round reports for one of the Targets it names:
// the phase of the Promotion it made for the Target, or that it made none.
export type RoundTargetPhase = PromotionPhase | typeof notAttempted;

// roundTargetPhases lists them in the order a summary of a round presents
// them: the Promotions' phases, then the Targets that got no Promotion.
export const roundTargetPhases: readonly RoundTargetPhase[] = [...promotionPhases, notAttempted];

// unreachedPhase is what a round reports for a Target it names and has made no
// Promotion for, going by the round's own phase. This is the one place that
// is decided, for every view of a round:
//
//   - A round still under way has yet to reach the Target, which is Pending.
//   - A round that ended without succeeding never will, so the Target was not
//     attempted. Reading it as Pending would leave it looking mid-promotion
//     for good; taking the round's own phase would report a failure on a
//     Target nothing ran against.
//   - A round that succeeded promoted to every Target it names, so a Target
//     with no Promotion on record is one whose record has not caught up.
export const unreachedPhase = (roundPhase?: string): RoundTargetPhase => {
  switch (roundPhase) {
    case 'Succeeded':
      return 'Succeeded';
    case 'Failed':
    case 'Errored':
    case 'Aborted':
      return notAttempted;
    default:
      return 'Pending';
  }
};

const notAttemptedPresentation: PromotionPhasePresentation = {
  ...neutral,
  icon: faCircleMinus,
  // The neutral a Target the round did nothing to is painted with wherever a
  // fleet is drawn, so a square and a bar segment agree.
  colorToken: 'colorTextQuaternary'
};

// getRoundTargetPhasePresentation is getPromotionPhasePresentation for what a
// round reports of a Target, which may be that it was not attempted.
export const getRoundTargetPhasePresentation = (phase?: string): PromotionPhasePresentation =>
  phase === notAttempted ? notAttemptedPresentation : getPromotionPhasePresentation(phase);

// roundTargetPhaseLabel is how a reader is told it: a Promotion's phase is
// its own name, and a Target that got no Promotion was "Not attempted". Lower
// case, it follows a count: "8 not attempted".
export const roundTargetPhaseLabel = (phase: RoundTargetPhase): string =>
  phase === notAttempted ? 'Not attempted' : phase;
