import { toJson } from '@bufbuild/protobuf';
import { Descriptions, Tabs } from 'antd';
import Link from 'antd/es/typography/Link';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ManifestPreview } from '@ui/features/common/manifest-preview';
import { FreightStatusList } from '@ui/features/freight/freight-status-list';
import { FreightTable } from '@ui/features/project/pipelines/freight/freight-table';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import { Freight, FreightSchema } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightDetailsProps = {
  freight: Freight;
};

export const FreightDetails = (props: FreightDetailsProps) => {
  const navigate = useNavigate();
  const freightCreatedAt = useGetFreightCreation(props.freight);

  return (
    <>
      <Tabs
        items={[
          {
            key: 'freight',
            label: 'Freight',
            children: (
              <>
                <Descriptions
                  column={2}
                  size='small'
                  bordered
                  items={[
                    {
                      label: 'alias',
                      children: props.freight?.alias
                    },
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

                <FreightStatusList freight={props.freight} />
              </>
            )
          },
          {
            key: 'manifest',
            label: 'YAML',
            children: (
              <ManifestPreview object={toJson(FreightSchema, props.freight)} height='500px' />
            )
          }
        ]}
      />
    </>
  );
};
