import { PromotionRequestSummary } from '@ui/gen/api/v2/models';

import {
  RoundTargetPhase,
  notAttempted,
  roundTargetPhaseLabel,
  roundTargetPhases
} from './promotion-phase';

// RoundSummary tallies the Targets a round names by what the round reports for
// each. The summary a round records counts its Promotions by phase; the
// Targets a round ended without promoting to have no Promotion to count, and
// are tallied here beside them.
export type RoundSummary = PromotionRequestSummary & { notAttempted?: number };

// RoundSegment is one phase's share of a round of fan-out: how many of its
// Targets are in that phase and what fraction of the round they make up.
export type RoundSegment = {
  phase: RoundTargetPhase;
  count: number;
  // percent is the segment's width as a share of the round, 0 to 100. The
  // percents of all segments sum to 100 (up to floating-point rounding).
  percent: number;
};

export type RoundProgress = {
  total: number;
  segments: RoundSegment[];
};

// summaryKey maps a phase to its field on RoundSummary. A Promotion's phase is
// counted under its own name in lower case, as the summary a round records
// has it.
export const summaryKey = (phase: RoundTargetPhase): keyof RoundSummary =>
  phase === notAttempted
    ? 'notAttempted'
    : (phase.toLowerCase() as Lowercase<typeof phase> & keyof RoundSummary);

// roundSegments turns a round's summary into ordered, non-empty segments for a
// progress bar. Phases with no Targets are omitted, so a round that entirely
// succeeded yields a single full-width segment. A summary with no Targets
// yields no segments and a total of zero; callers decide how to draw that.
export const roundSegments = (summary?: RoundSummary): RoundProgress => {
  const counts = roundTargetPhases.map((phase) => ({
    phase,
    count: Math.max(0, summary?.[summaryKey(phase)] ?? 0)
  }));
  const total = counts.reduce((sum, { count }) => sum + count, 0);
  if (!total) {
    return { total: 0, segments: [] };
  }
  return {
    total,
    segments: counts
      .filter(({ count }) => count > 0)
      .map(({ phase, count }) => ({ phase, count, percent: (count / total) * 100 }))
  };
};

// describeRound is the tooltip text for a round: one line per non-empty phase,
// in presentation order, so "3 succeeded, 1 errored" reads the same everywhere.
export const describeRound = (progress: RoundProgress): string => {
  if (!progress.total) {
    return 'No Targets in this round';
  }
  const parts = progress.segments.map(
    ({ phase, count }) => `${count} ${roundTargetPhaseLabel(phase).toLowerCase()}`
  );
  return `${parts.join(', ')} of ${progress.total} Target${progress.total === 1 ? '' : 's'}`;
};

// roundBadgePhase picks the one phase that should color a round's badge: the
// worst thing that happened, or else whether it is still moving. Anything that
// did not succeed outranks everything, since one failed Target is the fact a
// reader needs; movement outranks rest; a round that was cut short, by an
// abort or by ending before it reached every Target, is not one that
// succeeded; and a round with no children has no phase to speak of.
export const roundBadgePhase = (progress: RoundProgress): RoundTargetPhase | undefined => {
  if (!progress.total) {
    return undefined;
  }
  const present = new Set(progress.segments.map((segment) => segment.phase));
  if (present.has('Errored')) {
    return 'Errored';
  }
  if (present.has('Failed')) {
    return 'Failed';
  }
  if (present.has('Running')) {
    return 'Running';
  }
  if (present.has('Pending')) {
    return 'Pending';
  }
  if (present.has('Aborted')) {
    return 'Aborted';
  }
  if (present.has(notAttempted)) {
    return notAttempted;
  }
  return 'Succeeded';
};

// roundBadgeText is the figure a badge shows: how many of the round's Targets
// have succeeded so far, over how many there are.
export const roundBadgeText = (progress: RoundProgress): string => {
  const succeeded = progress.segments.find((s) => s.phase === 'Succeeded')?.count ?? 0;
  return `${succeeded}/${progress.total}`;
};
