import {
  isPromotionRequestPhaseTerminal,
  promotionRequestCompareFn
} from '@ui/features/stage/utils/promotion-request';
import { PromotionRequest, Stage, Target, V1LabelSelector } from '@ui/gen/api/v2/models';

import { isStageTargetAware } from '../pipelines/nodes/stage-meta-utils';

// UNLABELED is the group a Target falls into when it carries no value for the
// label key the view is grouped by.
export const UNLABELED = '(unlabeled)';

// Label keys Kargo stamps on resources for its own bookkeeping. They are not
// useful axes to group a fleet by, so the group-by control hides them.
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
// view can offer them as grouping axes without anyone declaring them.
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

// GROUP_BY_STAGE is the grouping axis that follows the rollout's own
// structure: one group per governing Stage, plus one for ungoverned Targets.
export const GROUP_BY_STAGE = 'stage';

// UNGOVERNED is the group for Targets no target-aware Stage selects.
export const UNGOVERNED = '(ungoverned)';

// FleetRow is one Target under one governing Stage. A Target governed by two
// Stages yields two rows; one governed by none yields a row with no Stage.
export type FleetRow = {
  key: string;
  target: Target;
  stage?: Stage;
  cell: TargetStageCell;
};

export type FleetGroup = { value: string; rows: FleetRow[] };

// fleetRows pairs every Target with each Stage that governs it. Rows sort by
// Target name, then Stage name.
export const fleetRows = (
  targets: Target[],
  stages: Stage[],
  requests: PromotionRequest[]
): FleetRow[] => {
  const rows: FleetRow[] = [];
  for (const target of [...targets].sort(byName)) {
    const targetName = target.metadata?.name || '';
    const governing = stages.filter((stage) => stageGovernsTarget(stage, target)).sort(byName);
    if (!governing.length) {
      rows.push({
        key: targetName,
        target,
        cell: { governed: false, included: false }
      });
      continue;
    }
    for (const stage of governing) {
      rows.push({
        key: `${targetName}/${stage.metadata?.name}`,
        target,
        stage,
        cell: targetStageCell(stage, target, requests)
      });
    }
  }
  return rows;
};

export type PhaseCounts = { total: number; byPhase: Record<string, number> };

// Severity ranks a row for ordering: rounds that went wrong come first, rounds
// still moving second, and settled or idle rows last.
export const SEVERITY_FAILED = 0;
export const SEVERITY_ACTIVE = 1;
export const SEVERITY_SETTLED = 2;

export const rowSeverity = (row: FleetRow): number => {
  switch (row.cell.phase) {
    case 'Failed':
    case 'Errored':
      return SEVERITY_FAILED;
    case 'Pending':
    case 'Running':
      return SEVERITY_ACTIVE;
    default:
      return SEVERITY_SETTLED;
  }
};

// groupSeverity is the worst severity among a group's rows, so a group with one
// failure sorts as failed.
export const groupSeverity = (group: FleetGroup): number =>
  group.rows.reduce((worst, row) => Math.min(worst, rowSeverity(row)), SEVERITY_SETTLED);

// groupRows buckets rows by Stage, by the value they carry for a label key, or
// not at all. Groups with trouble sort first, then by value, with the
// ungoverned or unlabeled bucket last among its peers; rows keep their order
// within a group.
export const groupRows = (rows: FleetRow[], key?: string): FleetGroup[] => {
  if (!key) {
    return rows.length ? [{ value: '', rows }] : [];
  }
  const last = key === GROUP_BY_STAGE ? UNGOVERNED : UNLABELED;
  const valueOf = (row: FleetRow) =>
    key === GROUP_BY_STAGE
      ? row.stage?.metadata?.name || UNGOVERNED
      : (row.target.metadata?.labels?.[key] ?? UNLABELED);
  const groups = new Map<string, FleetRow[]>();
  for (const row of rows) {
    const value = valueOf(row);
    groups.set(value, [...(groups.get(value) || []), row]);
  }
  return [...groups.entries()]
    .map(([value, grouped]) => ({ value, rows: grouped }))
    .sort((lhs, rhs) => {
      const bySeverity = groupSeverity(lhs) - groupSeverity(rhs);
      if (bySeverity !== 0) {
        return bySeverity;
      }
      if (lhs.value === last) {
        return 1;
      }
      if (rhs.value === last) {
        return -1;
      }
      return lhs.value.localeCompare(rhs.value);
    });
};

// rowPhaseCounts tallies the phases of governed rows, for a summary of a group
// or of the whole fleet. Rows a round has not reached count as Pending.
export const rowPhaseCounts = (rows: FleetRow[]): PhaseCounts => {
  const counts: PhaseCounts = { total: 0, byPhase: {} };
  for (const row of rows) {
    if (!row.cell.governed) {
      continue;
    }
    counts.total++;
    const phase = row.cell.phase || 'Pending';
    counts.byPhase[phase] = (counts.byPhase[phase] || 0) + 1;
  }
  return counts;
};

// requestForStage picks the PromotionRequest that best describes what the
// Stage is doing right now: the one it reports as current, else the one it
// reports as its last, else the newest it has at all -- the last two cover a
// Stage whose status has not caught up with its requests yet.
export const requestForStage = (
  stage: Stage,
  requests: PromotionRequest[]
): PromotionRequest | undefined => {
  const stageName = stage.metadata?.name;
  const forStage = requests.filter((request) => request.spec?.stage === stageName);
  const named = (name?: string) =>
    name ? forStage.find((request) => request.metadata?.name === name) : undefined;
  return (
    named(stage.status?.currentPromotionRequest?.name) ||
    named(stage.status?.lastPromotionRequest?.name) ||
    [...forStage].sort(promotionRequestCompareFn)[0]
  );
};

export type TargetStageCell = {
  // Whether the Stage's selectors currently select this Target.
  governed: boolean;
  // The round whose outcome the cell reports, if the Stage has had one.
  request?: PromotionRequest;
  // Whether that round named this Target. A Target that joined the Stage after
  // the round was created is governed but not included.
  included: boolean;
  // The child Promotion promoting to this Target in that round, once created.
  promotion?: string;
  // The phase to show. Comes from the child Promotion when there is one; from
  // the round itself when it ended before a child could be created; and is
  // Pending while the round has yet to reach this Target.
  phase?: string;
};

// targetStageCell resolves what one Target's cell under one Stage should show,
// from the Stage's most relevant PromotionRequest.
export const targetStageCell = (
  stage: Stage,
  target: Target,
  requests: PromotionRequest[]
): TargetStageCell => {
  const governed = stageGovernsTarget(stage, target);
  const request = requestForStage(stage, requests);
  const name = target.metadata?.name;
  const included = !!request?.spec?.targets?.some((named) => named.name === name);
  if (!request || !included) {
    return { governed, request, included };
  }
  const status = request.status?.targets?.find((named) => named.name === name);
  if (status) {
    return { governed, request, included, promotion: status.promotion, phase: status.phase };
  }
  const requestPhase = request.status?.phase;
  return {
    governed,
    request,
    included,
    phase: isPromotionRequestPhaseTerminal(requestPhase) ? requestPhase : 'Pending'
  };
};

// phaseCounts tallies the phases of the Targets a Stage governs, feeding the
// per-Stage progress summary. Targets the round has not named count as Pending.
export const phaseCounts = (
  stage: Stage,
  targets: Target[],
  requests: PromotionRequest[]
): PhaseCounts => {
  const counts: PhaseCounts = { total: 0, byPhase: {} };
  for (const target of targets) {
    const cell = targetStageCell(stage, target, requests);
    if (!cell.governed) {
      continue;
    }
    counts.total++;
    const phase = cell.phase || 'Pending';
    counts.byPhase[phase] = (counts.byPhase[phase] || 0) + 1;
  }
  return counts;
};
