import { Descriptions } from 'antd';

import { FreightTable } from '@ui/features/project/pipelines/freight/freight-table';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightDetailsProps = {
  freight: Freight;
};

export const FreightDetails = (props: FreightDetailsProps) => {
  const freightCreatedAt = useGetFreightCreation(props.freight);

  return (
    <>
      <Descriptions
        column={2}
        size='small'
        bordered
        title='Freight'
        items={[
          {
            label: 'id',
            children: props.freight?.metadata?.name
          },
          {
            label: 'uid',
            children: props.freight?.metadata?.uid
          },
          {
            label: 'created',
            children: `${freightCreatedAt.relative}${freightCreatedAt.relative && ' , on'} ${freightCreatedAt.abs}`
          }
        ]}
      />

      <FreightTable className='mt-5' freight={props.freight} />
    </>
  );
};
