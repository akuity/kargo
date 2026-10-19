import {
  PhaseFilter,
  PromotionPhase,
  isPromotionPhase,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { Freight, PromotionRequest, PromotionRequestSummary } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

// summaryKey maps a phase to its counter on PromotionRequestSummary, whose
// fields are the phases in lower case.
const summaryKey = (phase: PromotionPhase) =>
  phase.toLowerCase() as Lowercase<PromotionPhase> & keyof PromotionRequestSummary;

// requestPhase is the phase a PromotionRequest is in. A request the reconciler
// has not acted on yet records none, and is in effect Pending.
export const requestPhase = (request?: PromotionRequest): PromotionPhase => {
  const phase = request?.status?.phase;
  return isPromotionPhase(phase) ? phase : 'Pending';
};

// roundSummary is the per-phase tally of a round's child Promotions. A request
// that has not fanned out yet has no summary, in which case every Target it
// names is, in effect, pending.
export const roundSummary = (request?: PromotionRequest): PromotionRequestSummary =>
  request?.status?.summary ?? { pending: request?.spec?.targets?.length || 0 };

// TargetsTally is a round's outcome as the Targets column says it: a figure per
// phase, over how many Targets the round named.
export type TargetsTally = {
  total: number;
  parts: { phase: PromotionPhase; count: number }[];
};

// targetsTally reads "130 succeeded · 6 failed / 136" once rendered. Succeeded
// leads, because it is the figure a reader looks for, and it is kept even at
// zero so the column reads the same from row to row. Every other non-empty
// phase follows in presentation order, so a round still fanning out shows how
// far it has got rather than only what has succeeded.
export const targetsTally = (summary: PromotionRequestSummary): TargetsTally => {
  const counts = promotionPhases.map((phase) => ({
    phase,
    count: Math.max(0, summary[summaryKey(phase)] ?? 0)
  }));
  const total = counts.reduce((sum, { count }) => sum + count, 0);
  if (!total) {
    return { total: 0, parts: [] };
  }
  return { total, parts: counts.filter(({ phase, count }) => phase === 'Succeeded' || count > 0) };
};

// describeTargets is that same tally as one string, for a tooltip or a label.
export const describeTargets = (tally: TargetsTally): string => {
  if (!tally.total) {
    return 'no Targets';
  }
  const parts = tally.parts.map(({ phase, count }) => `${count} ${phase.toLowerCase()}`);
  return `${parts.join(' · ')} / ${tally.total}`;
};

// requestDurationMs is how long a round took, or how long it has been going.
// Clock skew between the control plane and the browser can put `now` behind the
// start, and showing a negative duration would be worse than showing none.
export const requestDurationMs = (
  request?: PromotionRequest,
  now: number = Date.now()
): number | undefined => {
  const started = parseDate(request?.status?.startedAt);
  if (!started) {
    return undefined;
  }
  const finished = parseDate(request?.status?.finishedAt);
  const elapsed = (finished?.getTime() ?? now) - started.getTime();
  return elapsed < 0 ? undefined : elapsed;
};

// chipPhases is the order the filter chips are offered in: what is still
// moving, then what finished well, then what did not.
const chipPhases: readonly PromotionPhase[] = [
  'Running',
  'Pending',
  'Succeeded',
  'Failed',
  'Errored',
  'Aborted'
];

// alwaysOffered phases keep their chip even at zero, so a reader can see that
// nothing is running, or that nothing has failed, rather than having to infer
// it from a chip that is not there. The rest appear only once they occur.
const alwaysOffered: readonly PromotionPhase[] = ['Running', 'Succeeded', 'Failed'];

export const phaseFilters = (requests: PromotionRequest[]): PhaseFilter[] => {
  const counts = requests.reduce<Partial<Record<PromotionPhase, number>>>((acc, request) => {
    const phase = requestPhase(request);
    acc[phase] = (acc[phase] || 0) + 1;
    return acc;
  }, {});
  return [
    { count: requests.length },
    ...chipPhases
      .filter((phase) => counts[phase] || alwaysOffered.includes(phase))
      .map((phase) => ({ phase, count: counts[phase] || 0 }))
  ];
};

// freightVersion names the version a piece of Freight carries, for the few
// characters beside its alias. Only Freight whose versioned artifacts -- its
// images, charts and other artifacts -- agree on a single version has one to
// name; anything else is ambiguous and goes unnamed rather than having one of
// its versions picked arbitrarily. Git commits are left out: a commit is
// identified rather than versioned, and counting one would leave most Freight
// ambiguous.
export const freightVersion = (freight?: Freight): string | undefined => {
  const versions = new Set(
    [
      ...(freight?.images || []).map((image) => image.tag),
      ...(freight?.charts || []).map((chart) => chart.version),
      ...(freight?.artifacts || []).map((artifact) => artifact.version)
    ].filter((version): version is string => !!version)
  );
  return versions.size === 1 ? [...versions][0] : undefined;
};
