import { faPencil, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table, Tag, Typography } from 'antd';

import { useModal } from '@ui/features/common/modal/use-modal';
import { useListProjectConfigMaps, useListSharedConfigMaps } from '@ui/gen/api/v2/core/core';
import { V1ConfigMap } from '@ui/gen/api/v2/models';

import { CreateConfigMapModal } from './create-config-map-modal';
import { DeleteConfigMapButton } from './delete-config-map-button';
import { EditConfigMapModal } from './edit-config-map-modal';

type Props = {
  // empty string means shared across all projects
  project?: string;
};

export const ConfigMaps = ({ project = '' }: Props) => {
  const sharedQuery = useListSharedConfigMaps({ query: { enabled: !project } });
  const projectQuery = useListProjectConfigMaps(project, { query: { enabled: !!project } });
  const listQuery = project ? projectQuery : sharedQuery;

  const actionModal = useModal();

  const { show: showCreateConfigMapModal } = useModal((p) => (
    <CreateConfigMapModal {...p} project={project} />
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
      <Table<V1ConfigMap>
        className='my-2'
        scroll={{ x: 'max-content' }}
        dataSource={listQuery.data?.data?.items || []}
        loading={listQuery.isLoading}
        pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
        size='small'
      >
        <Table.Column<V1ConfigMap> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<V1ConfigMap>
          title='Keys'
          dataIndex='data'
          render={(data) => {
            const configMapKeys = Object.keys(data || {});

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
        <Table.Column<V1ConfigMap>
          align='right'
          width={200}
          render={(_, record) => {
            if (record?.metadata?.labels?.['kargo.akuity.io/replicated-from']) {
              return (
                <Typography.Text type='secondary' italic>
                  Replicated
                </Typography.Text>
              );
            }
            return (
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
                <DeleteConfigMapButton project={project} name={record?.metadata?.name || ''} />
              </Flex>
            );
          }}
        />
      </Table>
    </Card>
  );
};
