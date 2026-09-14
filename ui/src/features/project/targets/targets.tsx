import { faBullseye } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Input, Select, Skeleton, Table, Tag, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { useWatchTargets } from '@ui/features/stage/tabs/fleet/use-watch-targets';
import { getColors } from '@ui/features/stage/utils';
import { useListStages, useListTargets } from '@ui/gen/api/v2/core/core';

import { useWatchStages } from '../pipelines/use-watch-stages';

import { StagePill } from './stage-pill';
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
          <Flex gap={8} wrap>
            {row.stages.map(({ stage, health }) => (
              <StagePill
                key={stage.metadata?.name}
                projectName={project}
                stage={stage}
                health={health}
                stageColorMap={stageColorMap}
              />
            ))}
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
            pagination={false}
            rowKey={(row) => row.target.metadata?.name || ''}
          />
        </Flex>
      ))}
    </div>
  );
};
