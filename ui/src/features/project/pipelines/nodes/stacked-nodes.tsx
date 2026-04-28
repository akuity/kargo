import { Handle, Position } from '@xyflow/react';
import { Button, Card, Typography } from 'antd';
import classNames from 'classnames';

import { useGraphContext } from '../context/graph-context';

import styles from './node-size-source-of-truth.module.less';

export const StackedNodeBody = (props: { id?: string; count?: number; onClick?: () => void }) => (
  <div
    id={props.id}
    className={classNames(styles['stacked-node-size'], 'relative nodrag cursor-default')}
  >
    <div
      className='absolute w-full h-full -top-1 -right-1 shadow-md rounded-md z-10'
      style={{ background: 'var(--app-bg-elevated)' }}
    />
    <div
      className='absolute w-full h-full -top-2 -right-2 shadow-md rounded-md'
      style={{ background: 'var(--app-bg-elevated)' }}
    />
    <Card className={classNames(styles['stacked-node-size'], 'relative z-20')}>
      <Button size='small' onClick={props.onClick}>
        <Typography.Text type='secondary'>
          Expand {props.count !== undefined ? `${props.count} stages` : 'stages'}
        </Typography.Text>
      </Button>
    </Card>
  </div>
);

export const StackedNodes = (props: {
  data: {
    // x stages
    value: number;
    id: string;
    parentNodeId: string;
  };
}) => {
  const graphContext = useGraphContext();

  return (
    <>
      <Handle position={Position.Left} type='target' style={{ top: '50%' }} />
      <StackedNodeBody
        count={props.data?.value}
        onClick={() => graphContext?.onUnstack(props.data.parentNodeId)}
      />
    </>
  );
};
