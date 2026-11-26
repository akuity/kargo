import type { HealthStats } from '@ui/gen/api/v2/models';

export type HealthSummary = {
  healthy: number;
  count: number;
  allHealthy: boolean;
};

// The API omits zero-valued fields, so a Project with no Stages (or
// Warehouses, or Targets) reports a stats block with no count at all, or no
// block. Treat anything missing as zero so the card reads "0/0" rather than
// "/".
export const healthSummary = (stats?: { count?: number; health?: HealthStats }): HealthSummary => {
  const count = stats?.count ?? 0;
  const healthy = stats?.health?.healthy ?? 0;
  return { healthy, count, allHealthy: count > 0 && healthy === count };
};

// healthTooltip spells out what a row's "healthy/count" ratio means, e.g.
// "1 of 4 Stages healthy", or says plainly that there are none.
export const healthTooltip = (summary: HealthSummary, kind: string): string => {
  if (summary.count === 0) {
    return `No ${kind}s`;
  }
  const noun = summary.count === 1 ? kind : `${kind}s`;
  return `${summary.healthy} of ${summary.count} ${noun} healthy`;
};
