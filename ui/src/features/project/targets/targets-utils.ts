import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Health, Stage, Target, V1LabelSelector } from '@ui/gen/api/v2/models';

// UNLABELED is the group a Target falls into when it carries no value for the
// label key the page is grouped by.
export const UNLABELED = '(unlabeled)';

// Label keys Kargo stamps on resources for its own bookkeeping. They are not
// useful axes to group Targets by, so the group-by control hides them.
const internalLabelPrefix = 'kargo.akuity.io/';

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

// stageGovernsTarget mirrors the controller's rule: a Stage governs a Target
// when the Target matches any of the Stage's selectors. A classic Stage governs
// none, and so does a target-aware Stage whose selector list is empty.
export const stageGovernsTarget = (stage: Stage, target: Target): boolean =>
  isStageTargetAware(stage) &&
  (stage.spec?.targets?.selectors || []).some((selector) =>
    matchesLabelSelector(selector, target.metadata?.labels || {})
  );

const byName = <T extends { metadata?: { name?: string } }>(lhs: T, rhs: T) =>
  (lhs.metadata?.name || '').localeCompare(rhs.metadata?.name || '');

// targetAwareStages returns the Stages that promote through Targets, by name.
export const targetAwareStages = (stages: Stage[]): Stage[] =>
  stages.filter((stage) => isStageTargetAware(stage)).sort(byName);

// labelKeys returns every label key present on the Targets, sorted, so the
// page can offer them as grouping axes without anyone declaring them.
export const labelKeys = (targets: Target[]): string[] => {
  const keys = new Set<string>();
  for (const target of targets) {
    for (const key of Object.keys(target.metadata?.labels || {})) {
      if (!key.startsWith(internalLabelPrefix)) {
        keys.add(key);
      }
    }
  }
  return [...keys].sort();
};

// GoverningStage is one Stage that governs a Target, with the Target's health
// under that Stage as the Target's own status records it.
export type GoverningStage = { stage: Stage; health?: Health };

export type TargetRow = { target: Target; stages: GoverningStage[] };

// targetRows pairs every Target with the Stages that govern it, both sorted by
// name. A Target no Stage governs still gets a row, with no Stages.
export const targetRows = (targets: Target[], stages: Stage[]): TargetRow[] => {
  const governing = targetAwareStages(stages);
  return [...targets].sort(byName).map((target) => ({
    target,
    stages: governing
      .filter((stage) => stageGovernsTarget(stage, target))
      .map((stage) => ({
        stage,
        health: target.status?.stages?.[stage.metadata?.name || '']?.health
      }))
  }));
};

export type TargetGroup = { value: string; rows: TargetRow[] };

// groupTargetRows buckets rows by the value they carry for a label key. With no
// key, every row lands in one group. Groups sort by value with the unlabeled
// bucket last; rows keep their order within a group.
export const groupTargetRows = (rows: TargetRow[], key?: string): TargetGroup[] => {
  if (!key) {
    return rows.length ? [{ value: '', rows }] : [];
  }
  const groups = new Map<string, TargetRow[]>();
  for (const row of rows) {
    const value = row.target.metadata?.labels?.[key] ?? UNLABELED;
    groups.set(value, [...(groups.get(value) || []), row]);
  }
  return [...groups.entries()]
    .sort(([lhs], [rhs]) => {
      if (lhs === UNLABELED) {
        return 1;
      }
      if (rhs === UNLABELED) {
        return -1;
      }
      return lhs.localeCompare(rhs);
    })
    .map(([value, grouped]) => ({ value, rows: grouped }));
};

// matchesSearch reports whether a row mentions the needle in its Target name,
// any label, or any governing Stage name.
export const matchesSearch = (row: TargetRow, needle: string): boolean => {
  const trimmed = needle.trim().toLowerCase();
  if (!trimmed) {
    return true;
  }
  const haystack = [
    row.target.metadata?.name || '',
    ...Object.entries(row.target.metadata?.labels || {}).map(([k, v]) => `${k}=${v}`),
    ...row.stages.map(({ stage }) => stage.metadata?.name || '')
  ]
    .join(' ')
    .toLowerCase();
  return haystack.includes(trimmed);
};
