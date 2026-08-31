import { PromotionRequest } from '@ui/gen/api/v2/models';

const terminalPhases = ['Succeeded', 'Failed', 'Errored'];

export const isPromotionRequestPhaseTerminal = (phase?: string) =>
  terminalPhases.includes(phase || '');

// promotionRequestCompareFn orders PromotionRequests newest first. Names embed
// a ULID, so within a Stage, name order is creation order.
export const promotionRequestCompareFn = (lhs: PromotionRequest, rhs: PromotionRequest) =>
  (rhs?.metadata?.name || '').localeCompare(lhs?.metadata?.name || '');

// blockingMessage returns the message a user needs to see when a
// PromotionRequest is not going to progress. The Ready condition is where the
// reason is recorded -- most commonly that fanning Freight out to Targets is
// not available in this installation.
export const blockingMessage = (promotionRequest?: PromotionRequest) => {
  const ready = promotionRequest?.status?.conditions?.find(
    (condition) => condition.type === 'Ready'
  );
  return ready?.status === 'False' && ready?.message ? ready.message : undefined;
};

export type PromotionRequestTargetRow = {
  name: string;
  promotion?: string;
  phase?: string;
};

// targetRows merges the Targets a PromotionRequest names (spec.targets, the
// resolved snapshot) with what has become of each (status.targets, written as
// child Promotions are created). A Target with no status entry simply has no
// child Promotion yet.
export const targetRows = (promotionRequest?: PromotionRequest): PromotionRequestTargetRow[] => {
  const byName = new Map<string, PromotionRequestTargetRow>();
  for (const target of promotionRequest?.spec?.targets || []) {
    if (target?.name) {
      byName.set(target.name, { name: target.name });
    }
  }
  for (const target of promotionRequest?.status?.targets || []) {
    if (target?.name) {
      byName.set(target.name, {
        name: target.name,
        promotion: target.promotion,
        phase: target.phase
      });
    }
  }
  return [...byName.values()];
};
