import { Stage } from '@ui/gen/api/v2/models';

import { TargetRow, scopeToStage } from './target-rows';
import { pipelineOrder } from './target-stages';

// UNLABELED is the group a Target falls into when it carries no value for the
// label key the page is grouped by.
export const UNLABELED = '(unlabeled)';

// NO_GROUPING is the value that turns grouping off. Label keys are never empty,
// so the empty string cannot collide with a real key.
export const NO_GROUPING = '';

// STAGE_GROUPING is the value that groups by governing Stage rather than by a
// label. Parentheses are not legal in a label key, so it cannot collide with
// one.
export const STAGE_GROUPING = '(stage)';

// UNGOVERNED is the group a Target falls into, grouped by Stage, when no Stage
// selects it.
export const UNGOVERNED = '(ungoverned)';

// The group-by row carries this many label keys before the rest fold into a
// dropdown.
const maxInlineKeys = 4;

// A key this long crowds that row on its own, so it folds into the dropdown
// ahead of its turn.
const maxKeyLength = 12;

// splitKeys picks which grouping axes sit on the row and which fold into the
// dropdown. The tail folds in, and so does a key long enough to crowd the row
// however early it comes -- unless every key is that long, when there is
// nothing to gain.
export const splitKeys = (keys: string[]): { inline: string[]; overflow: string[] } => {
  const short = keys.filter((key) => key.length <= maxKeyLength);
  const preferred = short.length ? short : keys;
  const inline = preferred.slice(0, maxInlineKeys);
  return { inline, overflow: keys.filter((key) => !inline.includes(key)) };
};

export type TargetGroup = { value: string; rows: TargetRow[] };

// groupTargetRows buckets rows by the value they carry for a label key. With no
// key, every row lands in one group. Groups sort by value with the unlabeled
// bucket last; rows keep their order within a group.
//
// Grouped by Stage, the buckets are different in kind: a Target governed by
// several Stages lands in each of their groups, read as that Stage sees it, so
// the groups can hold more rows between them than there are Targets. They come
// out in pipeline order, which is why this takes the Project's Stages, with the
// ungoverned last.
export const groupTargetRows = (
  rows: TargetRow[],
  key?: string,
  stages: Stage[] = []
): TargetGroup[] => {
  if (!key) {
    return rows.length ? [{ value: '', rows }] : [];
  }
  if (key === STAGE_GROUPING) {
    return groupByStage(rows, stages);
  }
  const groups = new Map<string, TargetRow[]>();
  for (const row of rows) {
    // `||`, not `??`: an empty label value is legal in Kubernetes but names no
    // group, so it belongs with the Targets that carry no value at all rather
    // than forming a group with a blank heading.
    const value = row.target.metadata?.labels?.[key] || UNLABELED;
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

const groupByStage = (rows: TargetRow[], stages: Stage[]): TargetGroup[] => {
  const groups = new Map<string, TargetRow[]>();
  const ungoverned: TargetRow[] = [];
  for (const row of rows) {
    if (!row.stages.length) {
      ungoverned.push(row);
    }
    for (const governing of row.stages) {
      const name = governing.stage.metadata?.name || '';
      groups.set(name, [...(groups.get(name) || []), scopeToStage(row, governing)]);
    }
  }
  // Only the Stages some row is governed by, so a search that leaves a Stage
  // nothing leaves it no card either -- just as with label values.
  const governing = [
    ...new Map(
      rows.flatMap((row) => row.stages).map(({ stage }) => [stage.metadata?.name || '', stage])
    ).values()
  ];
  return [
    ...pipelineOrder(governing, stages).map((stage) => ({
      value: stage.metadata?.name || '',
      rows: groups.get(stage.metadata?.name || '') || []
    })),
    ...(ungoverned.length ? [{ value: UNGOVERNED, rows: ungoverned }] : [])
  ];
};

// groupAxis names what the page is grouped by, in words.
export const groupAxis = (groupKey: string): string =>
  groupKey === STAGE_GROUPING ? 'Stage' : groupKey;

// groupHeading names what a card holds, for a header that prints the label key
// separately above it.
export const groupHeading = (group: TargetGroup, groupKey?: string): string => {
  if (!groupKey) {
    return 'All Targets';
  }
  if (group.value === UNGOVERNED) {
    return 'no Stage';
  }
  return group.value === UNLABELED ? `no ${groupKey} label` : group.value;
};

// groupTitle names a group in one line, for a heading with no room to print
// the key separately.
export const groupTitle = (group: TargetGroup, groupKey?: string): string => {
  if (!groupKey || group.value === UNLABELED || group.value === UNGOVERNED) {
    return groupHeading(group, groupKey);
  }
  return groupKey === STAGE_GROUPING ? `Stage ${group.value}` : `${groupKey}=${group.value}`;
};
