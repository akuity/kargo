import { visibleLabels } from './target-labels';
import { TargetRow } from './target-rows';

// Requirement is one term of a label selector: a key that must, or must not,
// carry a value.
export type Requirement = { key: string; negated: boolean; value: string };

// parseSelector reads the label selector shorthand kubectl accepts -- comma
// separated key=value and key!=value terms -- and returns nothing when the
// text is not one, so the caller can fall back to a free-text search. Keys are
// case sensitive, as Kubernetes labels are. A term whose value is empty does
// not count as one, which keeps a selector being typed from matching nothing
// the moment the `=` lands.
export const parseSelector = (text: string): Requirement[] | undefined => {
  const terms = text
    .split(',')
    .map((term) => term.trim())
    .filter(Boolean);
  if (!terms.length) {
    return undefined;
  }
  const requirements: Requirement[] = [];
  for (const term of terms) {
    const match = /^([^=!\s]+)\s*(!=|==|=)\s*(.*)$/.exec(term);
    // A term with nothing on the right is half-typed, not a filter for the
    // empty string: reading `tier=` as one empties the fleet on the way to
    // `tier=primary`, with nothing on screen to say why.
    if (!match || !match[3].trim()) {
      return undefined;
    }
    requirements.push({ key: match[1], negated: match[2] === '!=', value: match[3].trim() });
  }
  return requirements;
};

// matchesRequirements ANDs parsed requirements over a label set. A negated
// requirement holds for a Target that does not carry the key at all, as a
// NotIn expression does.
const matchesRequirements = (
  requirements: Requirement[],
  labels: Record<string, string> = {}
): boolean =>
  requirements.every(({ key, negated, value }) =>
    negated ? labels[key] !== value : labels[key] === value
  );

// matchesSearch matches a row against the search box: a label selector when the
// text parses as one, else free text over the name, the labels the page shows,
// and the governing Stages. A selector names a key in full, so it reaches
// Kargo's own labels too.
export const matchesSearch = (row: TargetRow, needle: string): boolean => {
  const trimmed = needle.trim();
  if (!trimmed) {
    return true;
  }
  const requirements = parseSelector(trimmed);
  if (requirements) {
    return matchesRequirements(requirements, row.target.labels || {});
  }
  const haystack = [
    row.target.name || '',
    ...visibleLabels(row.target).map(([key, value]) => `${key}=${value}`),
    ...row.stages.map(({ stage }) => stage.metadata?.name || '')
  ]
    .join(' ')
    .toLowerCase();
  return haystack.includes(trimmed.toLowerCase());
};
