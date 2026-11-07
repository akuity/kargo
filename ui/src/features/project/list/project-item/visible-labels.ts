// Label prefixes owned by Kubernetes, Kargo and common tooling (Helm, Argo CD).
// They describe how a Project got deployed, not what it is, so they would only
// crowd the tile.
const SYSTEM_PREFIX_SUFFIXES = [
  'kubernetes.io',
  'k8s.io',
  'kargo.akuity.io',
  'argoproj.io',
  'helm.sh'
];

const isSystemKey = (key: string): boolean => {
  const slash = key.indexOf('/');
  if (slash < 0) {
    return false;
  }
  const prefix = key.slice(0, slash);
  return SYSTEM_PREFIX_SUFFIXES.some((s) => prefix === s || prefix.endsWith(`.${s}`));
};

// visibleLabels returns the Project's user-facing labels as [key, value]
// pairs, sorted by key so every tile lists them in the same order.
export const visibleLabels = (labels?: Record<string, string>): [string, string][] =>
  Object.entries(labels ?? {})
    .filter(([key]) => !isSystemKey(key))
    .sort(([a], [b]) => a.localeCompare(b));

// fitCount returns how many leading tags fit in `available` pixels on one line.
// When not all of them fit, room is kept for the "+N" tag (plusWidth) that
// stands in for the rest. At least one tag is always shown - truncated by CSS
// if it is too wide - so a single long label never collapses into a bare "+1".
export const fitCount = (
  widths: number[],
  plusWidth: number,
  available: number,
  gap: number
): number => {
  const total = widths.reduce((sum, w) => sum + w, 0) + gap * Math.max(widths.length - 1, 0);
  if (total <= available) {
    return widths.length;
  }
  let used = 0;
  for (let i = 0; i < widths.length; i++) {
    const next = used + (i > 0 ? gap : 0) + widths[i];
    if (next + gap + plusWidth > available) {
      return Math.max(i, 1);
    }
    used = next;
  }
  return widths.length;
};
