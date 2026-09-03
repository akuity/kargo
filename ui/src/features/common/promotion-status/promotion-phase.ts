import {
  IconDefinition,
  faCancel,
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { theme } from 'antd';

// PromotionPhase mirrors PromotionPhase on the back end. A PromotionRequest's
// summary counts child Promotions by these same phases, so both the Promotion
// and PromotionRequest UIs present them with one shared vocabulary.
export type PromotionPhase = 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Errored' | 'Aborted';

export type PromotionPhasePresentation = {
  icon: IconDefinition;
  // iconColor is applied to a standalone FontAwesome icon.
  iconColor: string;
  // tagColor is an Ant Design Tag preset.
  tagColor: 'default' | 'processing' | 'success' | 'error';
  spin: boolean;
};

const neutral = { iconColor: 'aaa', tagColor: 'default', spin: false } as const;

const failed: PromotionPhasePresentation = {
  icon: faCircleExclamation,
  iconColor: theme.defaultSeed.colorError,
  tagColor: 'error',
  spin: false
};

export const promotionPhasePresentations: Record<PromotionPhase, PromotionPhasePresentation> = {
  Pending: { ...neutral, icon: faHourglassStart },
  Running: { ...neutral, icon: faCircleNotch, tagColor: 'processing', spin: true },
  Succeeded: {
    icon: faCircleCheck,
    iconColor: theme.defaultSeed.colorSuccess,
    tagColor: 'success',
    spin: false
  },
  Failed: failed,
  Errored: failed,
  Aborted: { ...neutral, icon: faCancel }
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
