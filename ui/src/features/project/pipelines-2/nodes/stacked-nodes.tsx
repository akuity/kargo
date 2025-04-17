import { Handle, Position } from '@xyflow/react';
import { Button, Card, Typography } from 'antd';
import classNames from 'classnames';

import { useGraphContext } from '../context/graph-context';

import styles from './node-size-source-of-truth.module.less';

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
      <Handle id={props.data.id} position={Position.Left} type='target' />
      <div className={classNames(styles['stacked-node-size'], 'relative')}>
        <div className='absolute w-full h-full -top-1 -right-1 bg-white shadow-md rounded-md z-10' />
        <div className='absolute w-full h-full -top-2 -right-2 bg-white shadow-md rounded-md' />
        <Card className={classNames(styles['stacked-node-size'], 'relative z-20')}>
          <Button size='small' onClick={() => graphContext?.onUnstack(props.data.parentNodeId)}>
            <Typography.Text type='secondary'>Expand {props.data?.value} stages</Typography.Text>
          </Button>
        </Card>
      </div>
    </>
  );
};
