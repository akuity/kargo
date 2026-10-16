import { Target, V1LabelSelector } from '@ui/gen/api/v2/models';

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
export const visibleLabels = (target: Target): [string, string][] =>
  Object.entries(target.metadata?.labels || {}).filter(([key]) => !isInternalLabel(key));

// matchesLabelSelector evaluates a Kubernetes label selector against a label
// set the way the API server does: matchLabels and matchExpressions are ANDed,
// and an empty selector matches everything.
export const matchesLabelSelector = (
  selector: V1LabelSelector | undefined,
  labels: Record<string, string> = {}
): boolean => {
  for (const [key, value] of Object.entries(selector?.matchLabels || {})) {
    if (labels[key] !== value) {
      return false;
    }
  }
  for (const requirement of selector?.matchExpressions || []) {
    const key = requirement.key || '';
    const present = Object.prototype.hasOwnProperty.call(labels, key);
    const values = requirement.values || [];
    switch (requirement.operator) {
      case 'In':
        if (!present || !values.includes(labels[key])) {
          return false;
        }
        break;
      case 'NotIn':
        if (present && values.includes(labels[key])) {
          return false;
        }
        break;
      case 'Exists':
        if (!present) {
          return false;
        }
        break;
      case 'DoesNotExist':
        if (present) {
          return false;
        }
        break;
      default:
        return false;
    }
  }
  return true;
};

// labelKeys returns every label key present on the Targets, sorted, so the
// page can offer them as grouping axes without anyone declaring them.
export const labelKeys = (targets: Target[]): string[] => {
  const keys = new Set<string>();
  for (const target of targets) {
    for (const [key] of visibleLabels(target)) {
      keys.add(key);
    }
  }
  return [...keys].sort();
};
