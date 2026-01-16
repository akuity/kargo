import { useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table, Tag } from 'antd';

import { useModal } from '@ui/features/common/modal/use-modal';
import { listConfigMaps } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ConfigMap } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { CreateConfigMapModal } from './create-config-map-modal';
import { DeleteConfigMapButton } from './delete-config-map-button';
import { EditConfigMapModal } from './edit-config-map-modal';

type Props = {
  systemLevel?: boolean;
  // empty string means shared across all projects
  project?: string;
};

export const ConfigMaps = ({ systemLevel = false, project = '' }: Props) => {
  const listProjectConfigMapsQuery = useQuery(listConfigMaps, { project, systemLevel });

  const actionModal = useModal();

  const { show: showCreateConfigMapModal } = useModal((p) => (
    <CreateConfigMapModal {...p} project={project} systemLevel={systemLevel} />
  ));

  return (
    <Card
      className='min-h-full'
      type='inner'
      title='ConfigMap'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => showCreateConfigMapModal()}>
          Add ConfigMap
        </Button>
      }
    >
      <Table
        className='my-2'
        scroll={{ x: 'max-content' }}
        dataSource={listProjectConfigMapsQuery.data?.configMaps || []}
        loading={listProjectConfigMapsQuery.isLoading}
        pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
        size='small'
      >
        <Table.Column<ConfigMap> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<ConfigMap>
          title='Keys'
          dataIndex='data'
          render={(data) => {
            const configMapKeys = Object.keys(data);

            if (!configMapKeys.length) {
              return <Tag color='red'>It looks like this configmap is empty.</Tag>;
            }

            return configMapKeys.map((key) => (
              <Tag key={key} color='blue'>
                {key}
              </Tag>
            ));
          }}
        />
        <Table.Column<ConfigMap>
          align='right'
          width={200}
          render={(_, record) => (
            <Flex justify='flex-end' gap={8}>
              <Button
                icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                color='default'
                variant='filled'
                size='small'
                onClick={() =>
                  actionModal.show((p) => (
                    <EditConfigMapModal {...p} configMap={record} project={project} />
                  ))
                }
              >
                Edit
              </Button>
              <DeleteConfigMapButton
                project={project}
                name={record?.metadata?.name || ''}
                systemLevel={systemLevel}
              />
            </Flex>
          )}
        />
      </Table>
    </Card>
  );
};
