import { faBullseye } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Input, Select, Skeleton, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageTag } from '@ui/features/common/stage-tag';
import { useWatchTargets } from '@ui/features/stage/tabs/fleet/use-watch-targets';
import { getColors } from '@ui/features/stage/utils';
import { useListStages, useListTargets } from '@ui/gen/api/v2/core/core';

import { useWatchStages } from '../pipelines/use-watch-stages';

import {
  TargetRow,
  UNLABELED,
  groupTargetRows,
  labelKeys,
  matchesSearch,
  targetAwareStages,
  targetRows
} from './targets-utils';

// The Select option that turns grouping off. Label keys are never empty, so
// the empty string cannot collide with a real key.
const NO_GROUPING = '';

// A fleet can run to hundreds of Targets, so tables paginate client-side: the
// whole list is already loaded and kept live, only the rendering is paged. The
// single ungrouped table offers a choice of page size; a group's table is one
// of many on the page, so it keeps a short fixed page and hides the pager when
// it fits.
const ungroupedPagination = {
  defaultPageSize: 20,
  pageSizeOptions: [20, 50, 100],
  showSizeChanger: true,
  hideOnSinglePage: true,
  showTotal: (total: number, range: [number, number]) =>
    `${range[0]}-${range[1]} of ${total} Targets`
};
const groupPagination = { pageSize: 10, hideOnSinglePage: true };

// A Target governed by many Stages would otherwise fill its row with tags;
// past this many, the rest fold into a "+N more" tag that lists them on hover.
const maxStageTags = 6;

/**
 * Targets lists a project's Targets and, for each, the Stages that govern it.
 * Each Stage is the coloured pill the pipeline paints it with, and opens the
 * Stage's drawer -- on its Fleet tab, where that Stage's view of the Target
 * lives. Governance is evaluated from each Stage's selectors, mirroring the
 * controller; grouping axes are whatever labels the Targets carry.
 */
export const Targets = () => {
  const { name: project = '' } = useParams();

  const stagesQuery = useListStages(project, { freightOrigins: [] });
  const targetsQuery = useListTargets(project, undefined);
  useWatchStages(project);
  useWatchTargets(project, undefined, !!targetsQuery.data);

  const [groupBy, setGroupBy] = useState<string>();
  const [search, setSearch] = useState('');

  const stages = useMemo(
    () => targetAwareStages(stagesQuery.data?.data?.items || []),
    [stagesQuery.data]
  );
  const targets = useMemo(() => targetsQuery.data?.data?.items || [], [targetsQuery.data]);
  const stageColorMap = useMemo(
    () => getColors(project, stagesQuery.data?.data?.items || []),
    [project, stagesQuery.data]
  );

  const keys = useMemo(() => labelKeys(targets), [targets]);
  const groupKey = groupBy === undefined ? undefined : groupBy || undefined;

  const rows = useMemo(
    () => targetRows(targets, stages).filter((row) => matchesSearch(row, search)),
    [targets, stages, search]
  );
  const groups = useMemo(() => groupTargetRows(rows, groupKey), [rows, groupKey]);

  if (stagesQuery.isLoading || targetsQuery.isLoading) {
    return (
      <div className='p-6'>
        <Skeleton active />
      </div>
    );
  }

  const columns: ColumnsType<TargetRow> = [
    {
      title: 'Target',
      key: 'target',
      render: (_, row) => (
        <Flex vertical gap={4}>
          <Typography.Text className='text-xs font-semibold'>
            <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
            {row.target.metadata?.name}
          </Typography.Text>
          <Flex gap={4} wrap>
            {Object.entries(row.target.metadata?.labels || {})
              .filter(([key]) => key !== groupKey)
              .map(([key, value]) => (
                <Tag key={key} className='m-0 text-xs'>
                  {key}={value}
                </Tag>
              ))}
          </Flex>
        </Flex>
      )
    },
    {
      title: 'Stages',
      key: 'stages',
      width: '50%',
      render: (_, row) =>
        row.stages.length ? (
          <Flex gap={8} wrap align='center'>
            {row.stages.slice(0, maxStageTags).map(({ stage, health }) => (
              <Link
                key={stage.metadata?.name}
                to={generatePath(paths.stage, {
                  name: project,
                  stageName: stage.metadata?.name || ''
                })}
              >
                <StageTag
                  stage={stage}
                  projectName={project}
                  stageColorMap={stageColorMap}
                  health={health}
                />
              </Link>
            ))}
            {row.stages.length > maxStageTags && (
              <Tooltip
                title={row.stages
                  .slice(maxStageTags)
                  .map(({ stage }) => stage.metadata?.name)
                  .join(', ')}
              >
                <Tag className='mb-2 text-xs'>+{row.stages.length - maxStageTags} more</Tag>
              </Tooltip>
            )}
          </Flex>
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            No Stage selects this Target
          </Typography.Text>
        )
    }
  ];

  return (
    <div className='p-6 flex flex-col gap-4 overflow-auto h-full'>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Flex vertical gap={4}>
          <Typography.Title level={4} className='!mb-0'>
            <FontAwesomeIcon icon={faBullseye} className='mr-2' />
            Targets
          </Typography.Title>
          <Typography.Text type='secondary' className='text-xs'>
            {targets.length} Target{targets.length === 1 ? '' : 's'} governed by {stages.length}{' '}
            Stage{stages.length === 1 ? '' : 's'}
          </Typography.Text>
        </Flex>
        <Flex gap={8} wrap>
          <Input.Search
            allowClear
            placeholder='Filter by name, label, or Stage'
            className='w-72'
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Select
            className='w-48'
            value={groupKey ?? NO_GROUPING}
            onChange={(value) => setGroupBy(value)}
            options={[
              { value: NO_GROUPING, label: 'No grouping' },
              ...keys.map((key) => ({ value: key, label: `Group by ${key}` }))
            ]}
          />
        </Flex>
      </Flex>

      {groups.length === 0 && (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={targets.length ? 'No Targets match' : 'This project has no Targets'}
        />
      )}

      {groups.map((group) => (
        <Flex key={group.value || 'all'} vertical gap={8}>
          {groupKey && (
            <Flex gap={8} align='center'>
              <Tag color={group.value === UNLABELED ? 'default' : 'blue'} className='m-0'>
                {group.value === UNLABELED ? `no ${groupKey} label` : `${groupKey}=${group.value}`}
              </Tag>
              <Typography.Text type='secondary' className='text-xs'>
                {group.rows.length} Target{group.rows.length === 1 ? '' : 's'}
              </Typography.Text>
            </Flex>
          )}
          <Table
            dataSource={group.rows}
            columns={columns}
            size='small'
            pagination={groupKey ? groupPagination : ungroupedPagination}
            rowKey={(row) => row.target.metadata?.name || ''}
          />
        </Flex>
      ))}
    </div>
  );
};
