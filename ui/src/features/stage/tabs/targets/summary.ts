import { HealthStatus, healthStatusToEnum } from '@ui/features/common/health-status/utils';
import { GoverningStage, TargetRow } from '@ui/features/project/targets/model/target-rows';
import { Target } from '@ui/gen/api/v2/models';

// governing is where a Target stands with the one Stage this tab is about.
// Rows here are built from that Stage's statuses alone, so there is never
// more than one pairing to pick from.
export const governing = (row: TargetRow): GoverningStage | undefined => row.stages[0];

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

// owningTargetSet names the TargetSet a Target was generated from. Targets may
// also be written by hand, and then nothing owns them.
export const owningTargetSet = (target: Target): string | undefined =>
  target.metadata?.ownerReferences?.find(({ kind }) => kind === 'TargetSet')?.name;
