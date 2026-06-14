import { Typography } from 'antd';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { getFreightProvenanceRows } from './freight-provenance-utils';

type Props = {
  freight: Freight;
  showAlias: boolean;
  age?: string;
};

export const FreightProvenanceRows = (props: Props) => {
  const rows = getFreightProvenanceRows(props.freight, {
    showAlias: props.showAlias,
    age: props.age
  });

  return (
    <div className='flex flex-col gap-0.5 text-left min-w-0 w-full py-1'>
      {rows.map((row) => (
        <div
          className='grid grid-cols-[52px_minmax(0,1fr)] gap-2 min-w-0 text-[10px]'
          key={row.key}
        >
          <Typography.Text type='secondary' className='uppercase text-[10px] leading-4'>
            {row.label}
          </Typography.Text>
          <Typography.Text
            className='font-mono text-[10px] leading-4 truncate'
            title={row.title || row.value}
          >
            {row.value}
          </Typography.Text>
        </div>
      ))}
    </div>
  );
};
