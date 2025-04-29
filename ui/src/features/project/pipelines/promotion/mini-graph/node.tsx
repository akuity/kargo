import { Handle, Position } from '@xyflow/react';
import { Card } from 'antd';

export const Node = (props: { data: { label: string; handles?: number } }) => {
  const handles = props.data.handles || 1;

  const arr = new Array(handles).fill(0);

  return (
    <>
      {arr.map((_, idx) => (
        <Handle
          key={idx}
          type='target'
          id={`${idx}`}
          position={Position.Left}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      ))}
      <Card>{props.data.label}</Card>
      {arr.map((_, idx) => (
        <Handle
          key={idx}
          type='source'
          id={`${idx}`}
          position={Position.Right}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      ))}
    </>
  );
};
