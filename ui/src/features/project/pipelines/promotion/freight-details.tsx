import { Descriptions } from 'antd';
import Link from 'antd/es/typography/Link';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { FreightTable } from '@ui/features/project/pipelines/freight/freight-table';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightDetailsProps = {
  freight: Freight;
};

export const FreightDetails = (props: FreightDetailsProps) => {
  const navigate = useNavigate();
  const freightCreatedAt = useGetFreightCreation(props.freight);

  if (!props.freight) {
    return null;
  }

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
            children: (
              <Link
                onClick={() =>
                  navigate(
                    generatePath(paths.freight, {
                      name: props.freight?.metadata?.namespace,
                      freightName: props.freight?.metadata?.name
                    })
                  )
                }
              >
                {props.freight?.metadata?.name}
              </Link>
            )
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
