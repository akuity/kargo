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
const failedPhases = ['Failed', 'Errored', 'Aborted'];

export const isPromotionRequestFailed = (phase?: string) => failedPhases.includes(phase || '');

// A round that succeeded can still have left Targets behind -- as many as its
// update strategy tolerates failing -- and those are as worth another go as
// the Targets of a round that failed outright.
export const isPromotionRequestRetryable = (phase?: string, unsuccessfulTargets = 0) =>
  isPromotionRequestFailed(phase) || (phase === 'Succeeded' && unsuccessfulTargets > 0);

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

// The Ready condition reason a round carries while it waits for its Stage to
// let it start. A Stage holds its next round for as long as the Freight it is
// on is still being verified, so a round may wait as long as a verification
// runs -- which is the round working as intended, not one in trouble.
const awaitingStageReason = 'AwaitingStage';

// RoundBlock explains a round that has not fanned out. Its severity tells a
// round that is only waiting its turn ('info') from one that is not going to
// progress as things stand ('warning').
export type RoundBlock = {
  severity: 'info' | 'warning';
  title: string;
  description: string;
};

// hasFannedOut reports whether a round got as far as its Targets: it recorded
// one, or a tally of them.
const hasFannedOut = (promotionRequest?: PromotionRequest): boolean =>
  !!promotionRequest?.status?.targets?.length || !!promotionRequest?.status?.summary;

// isVerifying reports whether the Freight a Stage is on has a verification
// that has yet to finish.
const isVerifying = (stage?: Stage): boolean =>
  (stage?.status?.freightHistory?.[0]?.verificationHistory || []).some(
    ({ phase }) => phase === 'Pending' || phase === 'Running'
  );

// roundBlock explains, in words meant for a person, why a PromotionRequest
// has not fanned out. A round that did fan out and then had children fail is
// not blocked -- its Targets show the outcome -- so a request that recorded
// any Target or summary yields nothing even if its Ready condition is False.
//
// A round waiting for its Stage to let it start is queued, not stuck, and is
// said to be so as a notice rather than as a warning. Given the Stage, the
// notice also says when what the round is waiting for is a verification of
// the Freight the Stage is on.
//
// Every other reason is a round that cannot progress. The Ready message is
// written for operators; the one case a user is likely to meet -- fan-out
// being an Enterprise feature -- gets plain language, and any other reason is
// passed through under a general heading.
export const roundBlock = (
  promotionRequest?: PromotionRequest,
  stage?: Stage
): RoundBlock | undefined => {
  if (hasFannedOut(promotionRequest)) {
    return undefined;
  }
  const ready = promotionRequest?.status?.conditions?.find(
    (condition) => condition.type === 'Ready'
  );
  if (ready?.status !== 'False') {
    return undefined;
  }
  if (ready.reason === awaitingStageReason) {
    return {
      severity: 'info',
      title: 'Next promotion queued',
      description: isVerifying(stage)
        ? 'The next promotion to this Stage is queued. It is waiting for verification of the ' +
          "Stage's current Freight to finish, and will start once the Stage lets it."
        : 'The next promotion to this Stage is queued and will start once the Stage lets it.'
    };
  }
  if (!ready.message) {
    return undefined;
  }
  if (ready.reason === enterpriseOnlyReason) {
    return {
      severity: 'warning',
      title: 'Fleet management is a Kargo Enterprise feature',
      description:
        'This Stage promotes to Targets, which this installation of Kargo does not support. ' +
        'Freight promoted to this Stage will not reach its Targets.'
    };
  }
  return {
    severity: 'warning',
    title: 'Promotion to this Stage cannot progress',
    description: ready.message
  };
};

// isRoundQueued reports whether a round has yet to start: it is Pending, has
// attempted no Target, and nothing stands in its way but its turn. Such a
// round has nothing to say about any Target yet, so what is known of the
// Targets is still what the round before it left. A round that cannot
// progress is not queued; it is blocked, and is shown as that.
export const isRoundQueued = (promotionRequest?: PromotionRequest): boolean => {
  if (!promotionRequest || hasFannedOut(promotionRequest)) {
    return false;
  }
  const phase = promotionRequest.status?.phase;
  if (phase && phase !== 'Pending') {
    return false;
  }
  return roundBlock(promotionRequest)?.severity !== 'warning';
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
