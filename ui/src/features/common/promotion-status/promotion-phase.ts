import {
  IconDefinition,
  faCancel,
  faCircleCheck,
  faCircleExclamation,
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

export const isPromotionPhase = (phase?: string): phase is PromotionPhase =>
  !!phase && phase in promotionPhasePresentations;

// getPromotionPhasePresentation treats an unset or unknown phase as Pending,
// which is what a Promotion is until the controller says otherwise.
export const getPromotionPhasePresentation = (phase?: string): PromotionPhasePresentation =>
  promotionPhasePresentations[isPromotionPhase(phase) ? phase : 'Pending'];
