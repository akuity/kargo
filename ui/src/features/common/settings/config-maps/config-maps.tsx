import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table, Tag } from 'antd';
import { stringify } from 'yaml';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteResource,
  listConfigMaps
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ConfigMap } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { UpsertConfigMapsModal } from './upsert-config-maps';

type Props = {
  systemLevel?: boolean;
  project?: string;
};

export const ConfigMaps = ({ systemLevel = false, project = '' }: Props) => {
  const confirm = useConfirmModal();

  const listProjectConfigMapsQuery = useQuery(listConfigMaps, { project, systemLevel });

  const deleteResourceMutation = useMutation(deleteResource, {
    onSuccess: () => listProjectConfigMapsQuery.refetch()
  });

  const configMaps = listProjectConfigMapsQuery.data?.configMaps || [];

  const actionModal = useModal();

  const deleteConfigMap = (record: ConfigMap) => {
    confirm({
      title: <>Delete ConfigMap</>,
      content: (
        <p>
          Are you sure you want to delete ConfigMap <b>{record?.metadata?.name}</b>?
        </p>
      ),
      onOk: () => {
        const textEncoder = new TextEncoder();

        deleteResourceMutation.mutate({
          manifest: textEncoder.encode(
            stringify({
              apiVersion: 'v1',
              kind: 'ConfigMap',
              ...record
            })
          )
        });
      }
    });
  };

  return (
    <Card
      className='flex-1'
      type='inner'
      title='ConfigMap'
      extra={
        <Button
          icon={<FontAwesomeIcon icon={faPlus} />}
          onClick={() =>
            actionModal.show((p) => (
              <UpsertConfigMapsModal
                {...p}
                project={project}
                onSuccess={() => {
                  listProjectConfigMapsQuery.refetch();
                }}
              />
            ))
          }
        >
          Add ConfigMap
        </Button>
      }
    >
      <Table
        className='my-2'
        scroll={{ x: 'max-content' }}
        key={configMaps.length}
        dataSource={configMaps}
        loading={listProjectConfigMapsQuery.isLoading}
        pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
        size='small'
        columns={[
          {
            title: 'Name',
            key: 'name',
            render: (_, record) => record?.metadata?.name
          },
          {
            title: 'Keys',
            key: 'keys',
            render: (_, record) => {
              const configMapKeys = Object.keys(record?.data);

              if (!configMapKeys.length) {
                return <Tag color='red'>It looks like this configmap is empty.</Tag>;
              }

              return configMapKeys.map((key) => (
                <Tag key={key} color='blue'>
                  {key}
                </Tag>
              ));
            }
          },
          {
            key: 'actions',
            width: '256px',
            render: (_, record) => (
              <Flex justify='flex-end'>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
                  onClick={() =>
                    actionModal.show((p) => (
                      <UpsertConfigMapsModal
                        {...p}
                        project={project}
                        editing={record?.metadata?.name}
                        onSuccess={() => listProjectConfigMapsQuery.refetch()}
                      />
                    ))
                  }
                >
                  Edit
                </Button>
                <Button
                  icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                  color='danger'
                  variant='filled'
                  size='small'
                  onClick={() => deleteConfigMap(record)}
                >
                  Delete
                </Button>
              </Flex>
            )
          }
        ]}
      />
    </Card>
  );
};
