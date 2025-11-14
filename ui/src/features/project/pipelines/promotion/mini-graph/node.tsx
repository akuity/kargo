import { Handle, Position } from '@xyflow/react';
import { Card } from 'antd';
import { ReactNode } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';

export const Node = (props: { data: { label: string; handles?: number; namespace: string } }) => {
  const handles = props.data.handles || 1;

  const arr = new Array(handles).fill(0);

  let label: ReactNode = props.data.label;

  if (props.data.namespace) {
    label = (
      <Link
        to={generatePath(paths.stage, {
          name: props.data.namespace,
          stageName: props.data.label
        })}
      >
        {label}
      </Link>
    );
  }

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
      <Card>{label}</Card>
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
