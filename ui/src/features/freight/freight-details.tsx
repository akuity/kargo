import { toJson } from '@bufbuild/protobuf';
import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faFile, faInfoCircle, faPencil } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Table, Tabs, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { useEffect, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Freight, FreightSchema } from '@ui/gen/api/v1alpha1/generated_pb';

import { Description } from '../common/description';
import { ManifestPreview } from '../common/manifest-preview';
import { useModal } from '../common/modal/use-modal';
import { getAlias } from '../common/utils';

import { ArtifactMetadata } from './artifact-metadata';
import { flattenFreightOrigin } from './flatten-freight-origin-utils';
import { FreightStatusList } from './freight-status-list';
import { UpdateFreightAliasModal } from './update-freight-alias-modal';

const CopyValue = (props: { value: string; label: string; className?: string }) => (
  <div className={classNames('flex items-center text-gray-500 font-mono', props.className)}>
    <span className='text-gray-400 mr-2 text-xs'>{props.label}</span>
    <Typography.Text copyable>{props.value}</Typography.Text>
  </div>
);
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

  return (
    <Drawer open={!!freight} onClose={onClose} width={'80%'} closable={false}>
      {freight && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between mb-4'>
            <div>
              <Typography.Title
                level={1}
                style={{ margin: 0, marginBottom: '0.5em' }}
                className='flex items-center'
              >
                {alias || freight.metadata?.name}
                {alias && (
                  <Tooltip placement='bottom' title='Edit Alias'>
                    <FontAwesomeIcon
                      icon={faPencil}
                      className='ml-2 text-gray-400 cursor-pointer text-sm hover:text-blue-500'
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
                    />
                  </Tooltip>
                )}
              </Typography.Title>
              {alias && freight?.metadata?.name && (
                <CopyValue label='NAME:' value={freight.metadata?.name} />
              )}
              <Description item={freight} loading={false} className='mt-2' />
            </div>

            {freight?.metadata?.uid && <CopyValue label='UID:' value={freight?.metadata?.uid} />}
          </div>
          <div className='flex flex-col flex-1'>
            <Tabs
              className='flex-1'
              defaultActiveKey='1'
              style={{ minHeight: '500px' }}
              items={[
                {
                  key: '1',
                  label: 'Details',
                  icon: <FontAwesomeIcon icon={faInfoCircle} />,
                  children: (
                    <>
                      <div className='mb-4'>
                        <div className='font-semibold mb-2 text-xs'>ARTIFACTS</div>
                        <Table
                          pagination={{
                            pageSize: 5
                          }}
                          dataSource={flattenFreightOrigin(freight)}
                          columns={[
                            {
                              title: 'Source',
                              render: (_, { type }) => {
                                let icon: IconProp = faGitAlt;

                                switch (type) {
                                  case 'helm':
                                    icon = faAnchor;
                                    break;
                                  case 'image':
                                    icon = faDocker;
                                    break;
                                }

                                return <FontAwesomeIcon icon={icon} />;
                              },
                              width: '5%'
                            },
                            {
                              title: 'Repo',
                              dataIndex: 'repoURL',
                              width: '30%'
                            },
                            {
                              title: 'Version',
                              render: (_, record) => {
                                switch (record.type) {
                                  case 'git':
                                    return record.id;
                                  case 'helm':
                                    return record.version;
                                  case 'image':
                                    return record.tag;
                                }
                              }
                            },
                            {
                              title: 'Metadata',
                              width: '600px',
                              render: (_, record) => {
                                return <ArtifactMetadata {...record} />;
                              }
                            }
                          ]}
                        />
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
                }
              ]}
            />
          </div>
        </div>
      )}
    </Drawer>
  );
};

export default FreightDetails;
