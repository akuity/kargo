import { Card, Empty, Table } from 'antd';
import { ColumnType } from 'antd/es/table';
import classNames from 'classnames';
import { ReactNode, useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useEventsWatcher } from '@ui/features/project/pipelines/graph/use-events-watcher';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { GroupSummaryTags } from '../nodes/stage-group-node';
import { getStageGroup, groupStages } from '../stage-groups';

import { actionColumn } from './columns/action-column';
import { healthColumn } from './columns/health-column';
import { lastPromotionColumn } from './columns/last-promotion-column';
import { phaseColumn } from './columns/phase-column';
import { stageColumn } from './columns/stage-column';
import { versionColumn } from './columns/version-column';

// A collapsible row standing for the stages that share a kargo.akuity.io/stage-group label.
type GroupRow = { group: string; stages: Stage[]; children: Stage[] };
type Row = Stage | GroupRow;

const isGroupRow = (row: Row): row is GroupRow => 'group' in row;

const rowName = (row: Row) => (isGroupRow(row) ? row.group : row.metadata?.name || '');

const groupRowKey = (group: string) => `stage-group/${group}`;

// Group rows show the name in the first column and the health roll-up in the health column.
const groupAware = (column: ColumnType<Stage>, index: number): ColumnType<Row> => ({
  ...(column as ColumnType<Row>),
  // the only sortable column sorts by stage name; a group row sorts by its group name
  sorter: column.sorter ? (a, b) => rowName(a).localeCompare(rowName(b)) : undefined,
  render: (value, row, rowIndex) => {
    if (!isGroupRow(row)) {
      return column.render?.(value, row, rowIndex) as ReactNode;
    }
    if (index === 0) {
      return (
        <b>
          {row.group} ({row.stages.length} stages)
        </b>
      );
    }
    if (column.title === 'Health') {
      return <GroupSummaryTags stages={row.stages} />;
    }
    return null;
  }
});

type PipelineListViewProps = {
  stages: Stage[];
  warehouses: WarehouseExpanded[];
  freights: Freight[];
  className?: string;
  project: string;
};

export const PipelineListView = (props: PipelineListViewProps) => {
  const actionContext = useActionContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  useEventsWatcher(
    props.project,
    undefined,
    freightTimelineControllerContext?.preferredFilter?.warehouses || []
  );

  const filteredStages = useMemo(() => {
    const filterWarehouses = freightTimelineControllerContext?.preferredFilter?.warehouses || [];
    const search = freightTimelineControllerContext?.stageSearch?.trim().toLowerCase() || '';

    let stages = props.stages;

    if (filterWarehouses.length) {
      stages = stages.filter((stage) =>
        stage.spec?.requestedFreight?.find((f) => filterWarehouses.includes(f?.origin?.name || ''))
      );
    }

    if (search) {
      stages = stages.filter((stage) => stage.metadata?.name?.toLowerCase().includes(search));
    }

    return stages;
  }, [
    freightTimelineControllerContext?.preferredFilter?.warehouses,
    freightTimelineControllerContext?.stageSearch,
    props.stages
  ]);

  const expandedGroups = freightTimelineControllerContext?.preferredFilter?.expandedStageGroups;
  const search = freightTimelineControllerContext?.stageSearch || '';

  // filteredStages already contains only search matches, so while searching every group is
  // shown expanded and no hit is hidden
  const { rows, expandedRowKeys } = useMemo(() => {
    const { members, collapsed } = groupStages(filteredStages, expandedGroups, search);
    const rows: Row[] = [];
    const seen = new Set<string>();
    for (const stage of filteredStages) {
      const group = getStageGroup(stage);
      if (!group) {
        rows.push(stage);
      } else if (!seen.has(group)) {
        seen.add(group);
        rows.push({ group, stages: members[group], children: members[group] });
      }
    }
    return {
      rows,
      expandedRowKeys: [...seen].filter((g) => !collapsed.has(g)).map(groupRowKey)
    };
  }, [filteredStages, expandedGroups, search]);

  const toggleGroup = (group: string, expand: boolean) => {
    const current = expandedGroups || [];
    freightTimelineControllerContext?.setPreferredFilter({
      ...freightTimelineControllerContext.preferredFilter,
      expandedStageGroups: expand
        ? [...current.filter((g) => g !== group), group]
        : current.filter((g) => g !== group)
    });
  };

  return (
    <Card className={classNames(props.className, 'm-2')} size='small'>
      <Table
        dataSource={rows}
        rowKey={(row) =>
          isGroupRow(row)
            ? groupRowKey(row.group)
            : `${row?.metadata?.name}-${row?.status?.observedGeneration}`
        }
        size='small'
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                props.stages.length ? 'No Stages match' : 'This project has no Stages yet'
              }
            />
          )
        }}
        expandable={{
          expandedRowKeys,
          onExpand: (expanded, row) => isGroupRow(row) && toggleGroup(row.group, expanded)
        }}
        columns={[
          stageColumn(),
          phaseColumn(),
          healthColumn(),
          versionColumn(),
          lastPromotionColumn(),
          actionColumn({
            onPromote: (stage) => actionContext?.actPromote(IAction.PROMOTE, stage)
          })
        ].map(groupAware)}
      />
    </Card>
  );
};
