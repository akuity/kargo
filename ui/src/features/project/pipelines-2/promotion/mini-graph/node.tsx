import { Handle, Position } from '@xyflow/react';
import { Card } from 'antd';
import { ReactNode } from 'react';

export const Node = (props: { data: { label: string; handles?: number } }) => {
  const handles = props.data.handles || 1;

  let TargetHandles: ReactNode;
  let SourceHandles: ReactNode;

  for (let i = 0; i < handles; i++) {
    TargetHandles = (
      <>
        {TargetHandles}
        <Handle
          type='target'
          id={`${i}`}
          position={Position.Left}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      </>
    );

    SourceHandles = (
      <>
        {SourceHandles}
        <Handle
          id={`${i}`}
          type='source'
          position={Position.Right}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      </>
    );
  }

  return (
    <>
      {TargetHandles}
      <Card>{props.data.label}</Card>
      <Handle
        type='source'
        position={Position.Right}
        style={{ backgroundColor: 'transparent', border: 'none' }}
      />
      {SourceHandles}
    </>
  );
};
