import { TargetSummary } from '@ui/gen/api/v2/models';

// The domain Kargo stamps its own bookkeeping under.
const internalLabelDomain = 'kargo.akuity.io';

// isInternalLabel reports whether a label key is Kargo's own. Kargo writes
// under its domain directly (`kargo.akuity.io/shard`) and under subdomains of
// it (`targetset.kargo.akuity.io/owner`), so the whole domain counts.
export const isInternalLabel = (key: string): boolean => {
  const domain = key.split('/')[0];
  return domain === internalLabelDomain || domain.endsWith(`.${internalLabelDomain}`);
};

// visibleLabels is a Target's labels with Kargo's own left out, as every
// surface that prints or searches them wants them.
export const visibleLabels = (target: TargetSummary): [string, string][] =>
  Object.entries(target.labels || {}).filter(([key]) => !isInternalLabel(key));

// labelKeys returns every label key present on the Targets, sorted, so the
// page can offer them as grouping axes without anyone declaring them.
export const labelKeys = (targets: TargetSummary[]): string[] => {
  const keys = new Set<string>();
  for (const target of targets) {
    for (const [key] of visibleLabels(target)) {
      keys.add(key);
    }
  }
  return [...keys].sort();
};
