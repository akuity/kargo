import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';

// The phases a PromotionRequest does not come back from, mirroring
// PromotionRequestPhase.IsTerminal on the back end. Aborted belongs here: a
// round someone cancelled is as finished as one that failed, and reading it as
// still running leaves every Target it named looking like it is mid-promotion.
const terminalPhases = ['Succeeded', 'Failed', 'Errored', 'Aborted'];

export const isPromotionRequestPhaseTerminal = (phase?: string) =>
  terminalPhases.includes(phase || '');

// A PromotionRequest that did not succeed can be retried by promoting the same
// Freight to the Stage again, which fans out a fresh request.
const retryablePhases = ['Failed', 'Errored', 'Aborted'];

export const isPromotionRequestRetryable = (phase?: string) =>
  retryablePhases.includes(phase || '');

// promotionRequestCompareFn orders PromotionRequests newest first. Names embed
// a ULID, so within a Stage, name order is creation order.
export const promotionRequestCompareFn = (lhs: PromotionRequest, rhs: PromotionRequest) =>
  (rhs?.metadata?.name || '').localeCompare(lhs?.metadata?.name || '');

// currentRound picks, out of a list of PromotionRequests, the one that best
// describes what a Stage is doing right now: the one its status names as
// current, else the one it names as last, else the newest it has at all -- the
// last two cover a Stage whose status has not caught up with its requests yet.
// Requests belonging to other Stages are ignored, so the list may be a whole
// Project's. A Stage that is not target-aware has no rounds, and callers
// filter those out before asking.
export const currentRound = (
  stage: Stage | undefined,
  requests: PromotionRequest[]
): PromotionRequest | undefined => {
  const stageName = stage?.metadata?.name || '';
  if (!stage || !stageName) {
    return undefined;
  }
  return roundOf(
    stage,
    requests.filter((request) => request.spec?.stage === stageName)
  );
};

// roundOf applies that precedence to one Stage's own requests, already
// separated out.
const roundOf = (stage: Stage, mine: PromotionRequest[]): PromotionRequest | undefined => {
  const named = (name?: string) =>
    name ? mine.find((request) => request.metadata?.name === name) : undefined;
  return (
    named(stage.status?.currentPromotionRequest?.name) ||
    named(stage.status?.lastPromotionRequest?.name) ||
    [...mine].sort(promotionRequestCompareFn)[0]
  );
};

// currentRounds resolves the round of many Stages at once, keyed by Stage name.
// It separates the requests by Stage in one pass, where asking currentRound per
// Stage would walk and copy the Project's whole list once for each of them.
export const currentRounds = (
  stages: Stage[],
  requests: PromotionRequest[]
): Record<string, PromotionRequest | undefined> => {
  const byStage = new Map<string, PromotionRequest[]>();
  for (const request of requests) {
    const stageName = request.spec?.stage || '';
    byStage.set(stageName, [...(byStage.get(stageName) || []), request]);
  }
  return Object.fromEntries(
    stages
      .filter((stage) => stage.metadata?.name)
      .map((stage) => {
        const stageName = stage.metadata?.name || '';
        return [stageName, roundOf(stage, byStage.get(stageName) || [])];
      })
  );
};

// blockingMessage returns the message a user needs to see when a
// PromotionRequest is not going to progress. The reconciler explains the
// current phase in status.message; a request written before that field
// existed recorded its reason on the Ready condition only -- most commonly that
// fanning Freight out to Targets is not available in this installation.
export const blockingMessage = (promotionRequest?: PromotionRequest) => {
  if (promotionRequest?.status?.message) {
    return promotionRequest.status.message;
  }
  const ready = promotionRequest?.status?.conditions?.find(
    (condition) => condition.type === 'Ready'
  );
  return ready?.status === 'False' && ready?.message ? ready.message : undefined;
};

// The Ready condition reason the PromotionRequest reconciler in this repository
// records when it declines to fan a request out, because doing so is a feature
// of Kargo Enterprise.
const enterpriseOnlyReason = 'EnterpriseOnlyFeature';

export type RoundBlock = { title: string; description: string };

// roundBlock explains, in words meant for a person, why a PromotionRequest
// never fanned out. A round that did fan out and then had children fail is
// not blocked -- its Targets show the outcome -- so a request that recorded
// any Target or summary yields nothing even if its Ready condition is False.
// The Ready message is written for operators; the one case a user is likely
// to meet -- fan-out being an Enterprise feature -- gets plain language, and
// any other reason is passed through under a general heading.
export const roundBlock = (promotionRequest?: PromotionRequest): RoundBlock | undefined => {
  if (promotionRequest?.status?.targets?.length || promotionRequest?.status?.summary) {
    return undefined;
  }
  const ready = promotionRequest?.status?.conditions?.find(
    (condition) => condition.type === 'Ready'
  );
  if (ready?.status !== 'False' || !ready.message) {
    return undefined;
  }
  if (ready.reason === enterpriseOnlyReason) {
    return {
      title: 'Fleet management is a Kargo Enterprise feature',
      description:
        'This Stage promotes to Targets, which this installation of Kargo does not support. ' +
        'Freight promoted to this Stage will not reach its Targets.'
    };
  }
  return { title: 'Promotion to this Stage cannot progress', description: ready.message };
};

type PromotionRequestTargetRow = {
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
