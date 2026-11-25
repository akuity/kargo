import { isStagePromoting } from '@ui/features/stage/utils/promotion-rollout';
import { Stage, Target, TargetStatusEntry, TargetStatusSummary } from '@ui/gen/api/v2/models';

import { StateCounts, toStateCounts, totalStateCount } from './target-state';

export type Figures = {
  // Targets by the state they show, each counted once.
  counts: StateCounts;
  // Targets any governing Stage has left on older Freight, whatever their
  // state.
  behind: number;
  // Every Target counted.
  total: number;
};

// fleetFigures reads the summary of a whole Project into what its tiles show.
//
// The server counts only the Targets some Stage governs. The page also lists
// the Targets no Stage governs, as never promoted -- no Stage has promoted to
// them -- so they are counted under NeverPromoted here, which keeps the tiles
// equal to a tally of the rows the page draws.
export const fleetFigures = (summary?: TargetStatusSummary): Figures => {
  const counts = toStateCounts(summary?.states);
  const total = summary?.targets || 0;
  const ungoverned = Math.max(total - totalStateCount(counts), 0);
  counts.NeverPromoted += ungoverned;
  return { counts, behind: summary?.behind || 0, total: totalStateCount(counts) };
};

// stageFigures reads one Stage's counts out of a summary.
export const stageFigures = (
  summary: TargetStatusSummary | undefined,
  stageName: string
): Figures => {
  const forStage = summary?.stages?.find(({ stage }) => stage === stageName);
  const counts = toStateCounts(forStage?.states);
  return { counts, behind: forStage?.behind || 0, total: totalStateCount(counts) };
};

// shouldPollStatuses decides whether Target statuses can still change by
// themselves: a promotion is on its way to one of them, or a Stage says it is
// promoting -- which covers a promotion the statuses have not caught up with.
// Otherwise they change only when a Stage moves on, which statusSignature
// notices.
export const shouldPollStatuses = (entries: TargetStatusEntry[], stages: Stage[]): boolean =>
  entries.some((entry) => entry.inFlight) || stages.some((stage) => isStagePromoting(stage));

// StatusRead is what deciding on another read needs to know of the last one:
// whether it failed, and when a read last succeeded (0 before one has).
export type StatusRead = {
  failed: boolean;
  updatedAt: number;
};

// shouldPollList decides whether to read the list of statuses again. A read
// that failed is tried again: nothing else retries it, and until one succeeds
// there is nothing to show.
export const shouldPollList = (
  list: StatusRead,
  entries: TargetStatusEntry[],
  stages: Stage[]
): boolean => list.failed || shouldPollStatuses(entries, stages);

// shouldPollSummary decides whether to read the summary again. It goes by the
// list's rule, and is also read once more after the list's latest read: the
// two are read apart from the same mirror, which trails the cluster, so a
// summary read just before the mirror caught up would otherwise keep counting
// a promotion the list has already seen finish.
export const shouldPollSummary = (
  summary: StatusRead,
  list: StatusRead,
  entries: TargetStatusEntry[],
  stages: Stage[]
): boolean =>
  summary.failed || shouldPollList(list, entries, stages) || summary.updatedAt < list.updatedAt;

// statusSignature names, as one string to compare, everything about a set of
// Stages that changes where their Targets stand: the round each last started
// or finished, the Freight each is on, and which Targets each governs. Stages
// are watched, so a change in this string is the sign the statuses are due
// another read, even while nothing is being polled.
export const statusSignature = (stages: Stage[]): string =>
  stages
    .map((stage) =>
      JSON.stringify([
        stage.metadata?.name || '',
        stage.status?.lastPromotionRollout?.name || '',
        stage.status?.currentPromotionRequest?.name || '',
        stage.status?.lastPromotionRequest?.name || '',
        stage.status?.lastPromotion?.name || '',
        stage.status?.freightHistory?.[0]?.id || '',
        stage.spec?.targets?.selectors || []
      ])
    )
    // Each part is JSON, which escapes any newline, so a newline cannot run
    // two of them together.
    .sort()
    .join('\n');

// targetsSignature names, as one string to compare, what about a set of
// Targets decides which Stages govern them: which Targets there are, and their
// labels. A Target added, removed or relabelled has statuses to gain or lose,
// and the statuses are due another read even while nothing is being polled.
export const targetsSignature = (targets: Target[]): string =>
  targets
    .map((target) =>
      JSON.stringify([
        target.metadata?.name || '',
        Object.entries(target.metadata?.labels || {})
          .map(([key, value]) => `${key}=${value}`)
          .sort()
      ])
    )
    .sort()
    .join('\n');
