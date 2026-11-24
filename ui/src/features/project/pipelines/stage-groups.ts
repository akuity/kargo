import { Stage } from '@ui/gen/api/v2/models';

// Stages carrying this label with the same value are shown as one collapsible
// group. Stages without the label are never grouped.
export const STAGE_GROUP_LABEL = 'kargo.akuity.io/stage-group';

export const getStageGroup = (stage: Stage): string | undefined =>
  stage?.metadata?.labels?.[STAGE_GROUP_LABEL] || undefined;

export type StageGroups = {
  // group name -> member stages, in input order
  members: Record<string, Stage[]>;
  // groups that are drawn as one node / one collapsed row
  collapsed: Set<string>;
};

// expanded: group names the user has expanded.
// search: a group with a stage matching the search is treated as expanded, so
// search hits are never hidden inside a collapsed group.
export const groupStages = (stages: Stage[], expanded: string[] = [], search = ''): StageGroups => {
  const members: Record<string, Stage[]> = {};
  for (const stage of stages) {
    const group = getStageGroup(stage);
    if (group) {
      (members[group] ||= []).push(stage);
    }
  }

  const needle = search.trim().toLowerCase();
  const collapsed = new Set<string>();
  for (const [group, groupStagesList] of Object.entries(members)) {
    const hasMatch =
      !!needle &&
      groupStagesList.some((stage) => stage.metadata?.name?.toLowerCase().includes(needle));
    if (!expanded.includes(group) && !hasMatch) {
      collapsed.add(group);
    }
  }

  return { members, collapsed };
};

export type GroupSummary = { total: number; healthy: number; unhealthy: number; other: number };

// Health roll-up shown on a collapsed group. Degraded counts as unhealthy;
// Progressing, Unknown and missing health are "other".
export const summarizeGroup = (stages: Stage[]): GroupSummary => {
  const summary: GroupSummary = { total: stages.length, healthy: 0, unhealthy: 0, other: 0 };
  for (const stage of stages) {
    switch (stage?.status?.health?.status) {
      case 'Healthy':
        summary.healthy++;
        break;
      case 'Unhealthy':
      case 'Degraded':
        summary.unhealthy++;
        break;
      default:
        summary.other++;
    }
  }
  return summary;
};
