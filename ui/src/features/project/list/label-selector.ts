// buildLabelSelector turns the selected "key=value" pairs into a Kubernetes
// label selector string. Multiple selected values for the same key are OR'd
// together via a set-based requirement, while different keys are AND'd.
export const buildLabelSelector = (pairs: string[]): string => {
  const valuesByKey = new Map<string, string[]>();
  for (const pair of pairs) {
    const idx = pair.indexOf('=');
    if (idx < 0) continue;
    const key = pair.slice(0, idx);
    const value = pair.slice(idx + 1);
    const values = valuesByKey.get(key) ?? [];
    values.push(value);
    valuesByKey.set(key, values);
  }

  return Array.from(valuesByKey.entries())
    .map(([key, values]) =>
      values.length === 1 ? `${key}=${values[0]}` : `${key} in (${values.join(',')})`
    )
    .join(',');
};
