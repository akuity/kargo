import { toJson } from '@bufbuild/protobuf';
import { faFile, faInfoCircle, faPencil } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Descriptions, Drawer, Space, Tabs, Typography } from 'antd';
import { useEffect, useState } from 'react';
import { Link, generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Freight, FreightSchema } from '@ui/gen/api/v1alpha1/generated_pb';
import { useGetFreightLinks } from '@ui/gen/api/v2/core/core';

import { DeepLinks } from '../common/deep-links';
import { Description } from '../common/description';
import { ManifestPreview } from '../common/manifest-preview';
import { useModal } from '../common/modal/use-modal';
import { getAlias } from '../common/utils';
import { FreightTable } from '../project/pipelines/freight/freight-table';

import { FreightMetadata } from './freight-metadata';
import { FreightStatusList } from './freight-status-list';
import { UpdateFreightAliasModal } from './update-freight-alias-modal';

export const FreightDetails = ({
  freight,
  refetchFreight
}: {
  freight?: Freight;
  refetchFreight: () => void;
}) => {
  const navigate = useNavigate();
  const { name: projectName } = useParams();
  const [alias, setAlias] = useState<string | undefined>();

  useEffect(() => {
    if (freight) {
      setAlias(getAlias(freight as Freight));
    }
  }, [freight]);

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));
  const { show } = useModal();
  const { freightTabs } = useExtensionsContext();

  const freightNameOrAlias = alias || freight?.metadata?.name;
  const { data: freightLinksData } = useGetFreightLinks(
    projectName || '',
    freightNameOrAlias || '',
    { query: { enabled: !!projectName && !!freightNameOrAlias } }
  );

  return (
    <Drawer
      open={!!freight}
      onClose={onClose}
      width='80%'
      title={alias || freight?.metadata?.name}
      extra={
        freight && (
          <Space size={16}>
            <DeepLinks links={freightLinksData?.data?.links ?? []} />
            {alias && (
              <Button
                icon={<FontAwesomeIcon icon={faPencil} />}
                onClick={() =>
                  show((p) => (
                    <UpdateFreightAliasModal
                      {...p}
                      freight={freight || undefined}
                      project={freight?.metadata?.namespace || ''}
                      onSubmit={(newAlias) => {
                        setAlias(newAlias);
                        refetchFreight();
                        p.hide();
                      }}
                    />
                  ))
                }
              >
                Edit Alias
              </Button>
            )}
          </Space>
        )
      }
    >
      {freight && (
        <div className='flex flex-col h-full'>
          <Description item={freight} loading={false} className='mb-6' />

          <div className='flex flex-col flex-1'>
            <Tabs
              className='flex-1 -mt-4'
              defaultActiveKey='1'
              style={{ minHeight: '500px' }}
              items={[
                {
                  key: '1',
                  label: 'Details',
                  icon: <FontAwesomeIcon icon={faInfoCircle} />,
                  children: (
                    <>
                      <div className='mb-8'>
                        <Descriptions
                          className='mb-5 max-w-4xl'
                          column={1}
                          bordered
                          size='small'
                          items={[
                            ...(alias && freight?.metadata?.name
                              ? [
                                  {
                                    key: 'name',
                                    label: 'Name',
                                    children: (
                                      <Typography.Text copyable>
                                        {freight.metadata.name}
                                      </Typography.Text>
                                    )
                                  }
                                ]
                              : []),
                            ...(freight?.origin?.name
                              ? [
                                  {
                                    key: 'origin',
                                    label: 'Origin',
                                    children: (
                                      <Link
                                        to={generatePath(paths.warehouse, {
                                          name: projectName,
                                          warehouseName: freight.origin.name
                                        })}
                                      >
                                        {freight.origin.name}
                                      </Link>
                                    )
                                  }
                                ]
                              : [])
                          ]}
                        />
                        <br />
                        <FreightMetadata freight={freight} className='mb-5' />
                        <FreightTable freight={freight} />
                      </div>
                      <FreightStatusList freight={freight} />
                    </>
                  )
                },
                {
                  key: '2',
                  label: 'Live Manifest',
                  icon: <FontAwesomeIcon icon={faFile} />,
                  className: 'h-full pb-2',
                  children: (
                    <ManifestPreview object={toJson(FreightSchema, freight)} height='900px' />
                  )
                },
                ...freightTabs.map((data, index) => ({
                  children: (
                    <data.component
                      projectName={projectName || ''}
                      freightName={freight?.metadata?.name || ''}
                    />
                  ),
                  key: String(data.label + index),
                  label: data.label,
                  icon: data.icon
                }))
              ]}
            />
          </div>
        </div>
      )}
    </Drawer>
  );
};

export default FreightDetails;
