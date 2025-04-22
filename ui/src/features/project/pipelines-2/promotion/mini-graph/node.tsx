import { Handle, Position } from '@xyflow/react';
import { Card } from 'antd';

export const Node = (props: { data: { label: string } }) => {
  return (
    <>
      <Handle
        type='target'
        position={Position.Left}
        style={{ backgroundColor: 'transparent', border: 'none' }}
      />
      <Card>{props.data.label}</Card>
      <Handle
        type='source'
        position={Position.Right}
        style={{ backgroundColor: 'transparent', border: 'none' }}
      />
    </>
  );
};
