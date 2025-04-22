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
          position={Position.Left}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      </>
    );

    SourceHandles = (
      <>
        {SourceHandles}
        <Handle
          type='source'
          position={Position.Right}
          style={{ backgroundColor: 'transparent', border: 'none' }}
        />
      </>
    );
  }

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
