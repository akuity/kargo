import { PromotionWindowKind } from '@ui/gen/api/v2/models';

export const occurrenceColors = (kind: PromotionWindowKind) =>
  kind === PromotionWindowKind.PromotionWindowKindAllow
    ? {
        dot: 'bg-emerald-500',
        event:
          'border-emerald-300 bg-emerald-50 text-emerald-950 hover:bg-emerald-100 dark:border-emerald-500/40 dark:bg-emerald-500/15 dark:text-emerald-50 dark:hover:bg-emerald-500/25'
      }
    : {
        dot: 'bg-rose-500',
        event:
          'border-rose-300 bg-rose-50 text-rose-950 hover:bg-rose-100 dark:border-rose-500/40 dark:bg-rose-500/15 dark:text-rose-50 dark:hover:bg-rose-500/25'
      };

export const disabledOccurrenceStyle = {
  backgroundImage:
    'repeating-linear-gradient(45deg, rgba(120,120,120,0.28) 0 4px, transparent 4px 9px)'
};
