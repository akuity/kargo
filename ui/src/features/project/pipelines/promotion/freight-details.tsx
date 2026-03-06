import { toJson } from '@bufbuild/protobuf';
import { Descriptions, TabsProps } from 'antd';
import Link from 'antd/es/typography/Link';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ManifestPreview } from '@ui/features/common/manifest-preview';
import TabsWithUrl from '@ui/features/common/tabs-with-url';
import { FreightMetadata } from '@ui/features/freight/freight-metadata';
import { FreightStatusList } from '@ui/features/freight/freight-status-list';
import { FreightTable } from '@ui/features/project/pipelines/freight/freight-table';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import { Freight, FreightSchema } from '@ui/gen/api/v1alpha1/generated_pb';

type FreightDetailsProps = {
  freight: Freight;
  additionalTabs?: TabsProps['items'];
};

export const FreightDetails = (props: FreightDetailsProps) => {
  const navigate = useNavigate();
  const freightCreatedAt = useGetFreightCreation(props.freight);

  if (!props.freight) {
    return null;
  }

  return (
    <>
      <TabsWithUrl
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

                <FreightMetadata className='mt-5' freight={props.freight} />

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
          },
          ...(props.additionalTabs || [])
        ]}
      />
    </>
  );
};
