import { Handle, Position } from '@xyflow/react';
import { Flex, Tag, Typography } from 'antd';
import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v2/models';

import { useGraphContext } from '../context/graph-context';
import { summarizeGroup } from '../stage-groups';

import { StackedNodeBody } from './stacked-nodes';

export const GroupSummaryTags = (props: { stages: Stage[] }) => {
  const summary = useMemo(() => summarizeGroup(props.stages), [props.stages]);

  return (
    <Flex gap={2} wrap className='text-[10px]'>
      {summary.healthy > 0 && <Tag color='success'>{summary.healthy} healthy</Tag>}
      {summary.unhealthy > 0 && <Tag color='error'>{summary.unhealthy} unhealthy</Tag>}
      {summary.other > 0 && <Tag>{summary.other} unknown</Tag>}
    </Flex>
  );
};

// One collapsed group of Stages that share the kargo.akuity.io/stage-group label.
export const StageGroupNode = (props: { data: { name: string; stages: Stage[] } }) => {
  const graphContext = useGraphContext();

  // edges carry the warehouse name as handle id, so the node needs one handle per warehouse
  const origins = useMemo(() => {
    const names = new Set<string>();
    for (const stage of props.data.stages || []) {
      for (const requested of stage.spec?.requestedFreight || []) {
        names.add(requested?.origin?.name || '');
      }
    }
    return [...names];
  }, [props.data.stages]);

  const handleStyle = {
    top: '50%',
    backgroundColor: 'transparent',
    border: 'none'
  };

  return (
    <>
      {origins.map((origin) => (
        <Handle
          key={`t-${origin}`}
          id={origin}
          type='target'
          position={Position.Left}
          style={{ ...handleStyle, left: 2 }}
        />
      ))}
      <StackedNodeBody
        count={props.data.stages?.length || 0}
        onClick={() => graphContext?.onExpandGroup(props.data.name)}
      >
        <Flex vertical gap={4} className='mb-2'>
          <Typography.Text strong>{props.data.name}</Typography.Text>
          <GroupSummaryTags stages={props.data.stages || []} />
        </Flex>
      </StackedNodeBody>
      {origins.map((origin) => (
        <Handle
          key={`s-${origin}`}
          id={origin}
          type='source'
          position={Position.Right}
          style={{ ...handleStyle, right: 4 }}
        />
      ))}
    </>
  );
};
