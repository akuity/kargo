import { HealthStatus, healthStatusToEnum } from '@ui/features/common/health-status/utils';
import { GoverningStage, TargetRow } from '@ui/features/project/targets/model/target-rows';
import { PromotionRequest, Target } from '@ui/gen/api/v2/models';

// governing is what the one Stage this tab is about observes of a Target.
// Rows here are built against that Stage alone, so there is never more than
// one pairing to pick from; a Target whose labels stop matching has none.
export const governing = (row: TargetRow): GoverningStage | undefined => row.stages[0];

export type RoundTally = {
  // Targets the latest round named, and how it went for each of them.
  inRound: number;
  succeeded: number;
  failed: number;
  inFlight: number;
  aborted: number;
  // Targets the round never named, having joined the Stage after it began.
  notInRound: number;
};

// roundTally reads the Stage's latest round back from the rows themselves, so
// the figures on the tile and the squares below it are one count and cannot
// disagree.
export const roundTally = (rows: TargetRow[]): RoundTally => {
  const tally: RoundTally = {
    inRound: 0,
    succeeded: 0,
    failed: 0,
    inFlight: 0,
    aborted: 0,
    notInRound: 0
  };
  for (const row of rows) {
    switch (governing(row)?.phase) {
      case 'Succeeded':
        tally.inRound++;
        tally.succeeded++;
        break;
      case 'Failed':
      case 'Errored':
        tally.inRound++;
        tally.failed++;
        break;
      case 'Pending':
      case 'Running':
        tally.inRound++;
        tally.inFlight++;
        break;
      case 'Aborted':
        tally.inRound++;
        tally.aborted++;
        break;
      default:
        tally.notInRound++;
    }
  }
  return tally;
};

export type HealthTally = {
  // Targets something has actually assessed. Targets are not health-checked
  // yet in most installations, so this is commonly zero.
  assessed: number;
  // The statuses those assessments found, commonest first. These count the
  // assessed Targets and no others, so they sum to `assessed`.
  byStatus: { status: string; count: number }[];
  // Targets nothing has assessed, counted apart so a reader is never given a
  // breakdown that does not add up to the figure above it.
  unassessed: number;
};

export const healthTally = (rows: TargetRow[]): HealthTally => {
  const counts = new Map<string, number>();
  let unassessed = 0;
  for (const row of rows) {
    const status = healthStatusToEnum(governing(row)?.health?.status);
    if (status === HealthStatus.UNDEFINED) {
      unassessed++;
      continue;
    }
    counts.set(status, (counts.get(status) || 0) + 1);
  }
  return {
    assessed: rows.length - unassessed,
    byStatus: [...counts.entries()]
      .map(([status, count]) => ({ status, count }))
      .sort((lhs, rhs) => rhs.count - lhs.count || lhs.status.localeCompare(rhs.status)),
    unassessed
  };
};

// assessedHealth reports whether anything has assessed the health of any
// Target here. Targets are health-checked from the directives a successful
// Promotion leaves behind, so a Stage whose steps produce none never has any
// health to report -- and a surface that would paint one leaves it out rather
// than draw a whole fleet of Unknowns.
export const assessedHealth = (rows: TargetRow[]): boolean =>
  rows.some((row) => healthStatusToEnum(governing(row)?.health?.status) !== HealthStatus.UNDEFINED);

// neverPromoted counts the Targets this Stage has never landed Freight on. A
// Target with Freight from an earlier round still counts as promoted, however
// far behind that Freight now is.
export const neverPromoted = (rows: TargetRow[]): number =>
  rows.filter((row) => !row.freight.length).length;

// childPromotionName is the Promotion a round created to promote to one
// Target, once it has created one. It is how a Target reaches the steps and
// logs of its own promotion rather than the round's summary of them.
export const childPromotionName = (
  round: PromotionRequest | undefined,
  target: string
): string | undefined => round?.status?.targets?.find(({ name }) => name === target)?.promotion;

// owningTargetSet names the TargetSet that manages a Target. Every Target has
// one, which it names in its spec; nothing is named until the Target is read.
export const owningTargetSet = (target?: Target): string | undefined =>
  target?.spec?.targetSet || undefined;
