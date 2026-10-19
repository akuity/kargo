import { PromotionStepStatus } from '@ui/features/common/promotion-directive-step-status/promotion-step-status';
import {
  PhaseFilter,
  PromotionPhase,
  isPromotionPhase
} from '@ui/features/common/promotion-status/promotion-phase';
import {
  Promotion,
  PromotionRequest,
  PromotionRequestSummary,
  Target
} from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { targetRows } from '../utils/promotion-request';

// ChildPromotion is one Target of a round: the Promotion promoting Freight to
// it, how that went, and the labels the Target carries -- which are what a
// reader slices a round's failures by.
export type ChildPromotion = {
  target: string;
  promotion?: Promotion;
  phase: PromotionPhase;
  labels: Record<string, string>;
};

// severity ranks a child for ordering: what went wrong first, what is still
// moving next, what is settled last. The same ranking the Fleet tab uses, so a
// reader meets the fleet in the same order wherever they look at it.
const severity = (phase: PromotionPhase): number => {
  switch (phase) {
    case 'Failed':
    case 'Errored':
      return 0;
    case 'Running':
    case 'Pending':
      return 1;
    default:
      return 2;
  }
};

/**
 * childPromotions pairs every Target a round named with its child Promotion
 * and that Target's labels.
 *
 * The round's own status records a phase per Target, written as the reconciler
 * reaches each one; the child Promotion carries the detail -- its steps, its
 * message, when it finished. Where the two disagree the Promotion wins, since
 * the round's copy is a snapshot taken when it was last reconciled.
 */
export const childPromotions = (
  request?: PromotionRequest,
  promotions: Promotion[] = [],
  targets: Target[] = []
): ChildPromotion[] => {
  const byName = new Map(
    promotions.map((promotion) => [promotion.metadata?.name || '', promotion])
  );
  const labelsOf = new Map(
    targets.map((target) => [target.metadata?.name || '', target.metadata?.labels || {}])
  );

  return targetRows(request)
    .map<ChildPromotion>((row) => {
      const promotion = row.promotion ? byName.get(row.promotion) : undefined;
      const phase = promotion?.status?.phase ?? row.phase;
      return {
        target: row.name,
        promotion,
        phase: isPromotionPhase(phase) ? phase : 'Pending',
        labels: labelsOf.get(row.name) || {}
      };
    })
    .sort(
      (lhs, rhs) =>
        severity(lhs.phase) - severity(rhs.phase) || lhs.target.localeCompare(rhs.target)
    );
};

// childrenSummary tallies the children by phase, so the round's bar, its grid
// and its table are all drawn from the same counts and cannot disagree.
export const childrenSummary = (children: ChildPromotion[]): PromotionRequestSummary => {
  const summary: PromotionRequestSummary = {};
  for (const child of children) {
    const key = child.phase.toLowerCase() as keyof PromotionRequestSummary;
    summary[key] = (summary[key] || 0) + 1;
  }
  return summary;
};

// stepStatuses is the per-step outcome of one child Promotion, one entry per
// step it was given. A step with no status never ran: the Promotion stopped
// before reaching it.
export const stepStatuses = (promotion?: Promotion): (string | undefined)[] => {
  const executed = promotion?.status?.stepExecutionMetadata || [];
  const count = Math.max(promotion?.spec?.steps?.length || 0, executed.length);
  return Array.from({ length: count }, (_, index) => executed[index]?.status);
};

const failedStepStatuses: readonly string[] = [
  PromotionStepStatus.FAILED,
  PromotionStepStatus.ERRORED,
  PromotionStepStatus.ABORTED
];

// failedStepName names the step a Promotion stopped on, by the alias its author
// gave it and by position otherwise. Steps are read from the end: a Promotion
// that retried a step before giving up records the giving up last.
export const failedStepName = (promotion?: Promotion): string | undefined => {
  const executed = promotion?.status?.stepExecutionMetadata || [];
  for (let index = executed.length - 1; index >= 0; index--) {
    if (failedStepStatuses.includes(executed[index]?.status || '')) {
      return executed[index].alias || `step ${index + 1}`;
    }
  }
  return undefined;
};

// didNotSucceed is a child that failed on its own terms. Aborted is left out:
// a cancelled Promotion says something about the operator, not the Target.
const didNotSucceed = (child: ChildPromotion) =>
  child.phase === 'Failed' || child.phase === 'Errored';

// FailureTally is one line of the "where it failed" panel: what failed, how
// many times, and -- where the question means anything -- out of how many.
export type FailureTally = { key: string; failed: number; total?: number };

// failuresByStep counts the children that stopped on each step, worst first. It
// answers whether a round failed in one place or came apart in several.
export const failuresByStep = (children: ChildPromotion[]): FailureTally[] => {
  const counts = new Map<string, number>();
  for (const child of children.filter(didNotSucceed)) {
    const step = failedStepName(child.promotion);
    if (step) {
      counts.set(step, (counts.get(step) || 0) + 1);
    }
  }
  return [...counts]
    .map(([key, failed]) => ({ key, failed }))
    .sort((lhs, rhs) => rhs.failed - lhs.failed || lhs.key.localeCompare(rhs.key));
};

const labelKeyOf = (label: string) => label.slice(0, label.indexOf('='));

/**
 * failuresByLabel counts failures against each label value the round's Targets
 * carry, over how many of them carry it. It answers whether the trouble sits in
 * one slice of the fleet -- a tier, a region, a cloud -- or is spread across it.
 *
 * Values of one label key stay together, and keys are ordered by their worst
 * value, so a reader compares like with like: "tier=secondary 5/68, tier=primary
 * 1/68" says the trouble is in one tier, which interleaving every key by count
 * would bury.
 */
export const failuresByLabel = (children: ChildPromotion[], limit = 6): FailureTally[] => {
  const totals = new Map<string, number>();
  const failures = new Map<string, number>();
  for (const child of children) {
    const failed = didNotSucceed(child);
    for (const [key, value] of Object.entries(child.labels)) {
      const label = `${key}=${value}`;
      totals.set(label, (totals.get(label) || 0) + 1);
      if (failed) {
        failures.set(label, (failures.get(label) || 0) + 1);
      }
    }
  }

  const worstOfKey = new Map<string, number>();
  for (const [label, failed] of failures) {
    const key = labelKeyOf(label);
    worstOfKey.set(key, Math.max(worstOfKey.get(key) || 0, failed));
  }

  return [...failures]
    .map(([key, failed]) => ({ key, failed, total: totals.get(key) || 0 }))
    .sort((lhs, rhs) => {
      const lhsKey = labelKeyOf(lhs.key);
      const rhsKey = labelKeyOf(rhs.key);
      if (lhsKey !== rhsKey) {
        return (
          (worstOfKey.get(rhsKey) || 0) - (worstOfKey.get(lhsKey) || 0) ||
          lhsKey.localeCompare(rhsKey)
        );
      }
      return rhs.failed - lhs.failed || lhs.key.localeCompare(rhs.key);
    })
    .slice(0, limit);
};

// firstFailure is the child that failed earliest, which is where a reader looks
// when a round goes wrong: the failures that follow are often the same one
// repeated. A failure with no finish time cannot be placed in the order and
// yields to one that can.
export const firstFailure = (children: ChildPromotion[]): ChildPromotion | undefined => {
  const finishedAt = (child: ChildPromotion) =>
    parseDate(child.promotion?.status?.finishedAt)?.getTime() ?? Number.MAX_SAFE_INTEGER;
  return children.filter(didNotSucceed).sort((lhs, rhs) => finishedAt(lhs) - finishedAt(rhs))[0];
};

// conditionSummary reports the round's Ready condition the way the API writes
// it: "Ready=False · PromotionsFailed".
export const conditionSummary = (request?: PromotionRequest): string | undefined => {
  const ready = request?.status?.conditions?.find((condition) => condition.type === 'Ready');
  if (!ready?.status) {
    return undefined;
  }
  return ready.reason ? `Ready=${ready.status} · ${ready.reason}` : `Ready=${ready.status}`;
};

// matchesChild reports whether a child matches a free-text needle against its
// Target's name and its labels as key=value, case-insensitively. An empty
// needle matches everything. Mirrors the Fleet tab's Target filter.
export const matchesChild = (child: ChildPromotion, needle: string): boolean => {
  const text = needle.trim().toLowerCase();
  if (!text) {
    return true;
  }
  const haystack = [child.target, ...Object.entries(child.labels).map(([k, v]) => `${k}=${v}`)]
    .join(' ')
    .toLowerCase();
  return haystack.includes(text);
};

// childChipPhases is the order a round's own Promotions are offered in. It
// leads with what broke, unlike the list of rounds, which leads with what is
// moving: a reader opens one round to find out what went wrong in it.
const childChipPhases: readonly PromotionPhase[] = [
  'Failed',
  'Errored',
  'Running',
  'Pending',
  'Succeeded',
  'Aborted'
];

// Every phase but Aborted keeps its chip at zero, so the round's shape can be
// read off the chips alone. Aborted appears only once something was cancelled.
const childAlwaysOffered: readonly PromotionPhase[] = [
  'Failed',
  'Errored',
  'Running',
  'Pending',
  'Succeeded'
];

// childPhaseFilters builds the chips above a round's Promotions: All, then one
// per phase, each with how many of the round's children are in it.
export const childPhaseFilters = (children: ChildPromotion[]): PhaseFilter[] => {
  const counts = children.reduce<Partial<Record<PromotionPhase, number>>>((acc, child) => {
    acc[child.phase] = (acc[child.phase] || 0) + 1;
    return acc;
  }, {});
  return [
    { count: children.length },
    ...childChipPhases
      .filter((phase) => counts[phase] || childAlwaysOffered.includes(phase))
      .map((phase) => ({ phase, count: counts[phase] || 0 }))
  ];
};
