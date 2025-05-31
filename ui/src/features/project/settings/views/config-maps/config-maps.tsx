import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Space, Table, Tag } from 'antd';
import { useParams } from 'react-router-dom';
import { stringify } from 'yaml';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteResource,
  listProjectConfigMaps
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ConfigMap } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { UpsertConfigMaps } from './upsert-config-maps';

export const ConfigMaps = () => {
  const { name } = useParams();

  const confirm = useConfirmModal();

  const listProjectConfigMapsQuery = useQuery(listProjectConfigMaps, { project: name });

  const deleteResourceMutation = useMutation(deleteResource, {
    onSuccess: () => listProjectConfigMapsQuery.refetch()
  });

  const configMaps = listProjectConfigMapsQuery.data?.configMaps || [];

  const actionModal = useModal();

  const deleteConfigMap = (record: ConfigMap) => {
    confirm({
      title: (
        <Flex align='center'>
          <FontAwesomeIcon icon={faTrash} className='mr-2' />
          Delete ConfigMap
        </Flex>
      ),
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
              <UpsertConfigMaps
                {...p}
                project={name || ''}
                onSuccess={() => {
                  p.hide();
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
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
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
            render: (_, record) => (
              <Space>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
                  onClick={() =>
                    actionModal.show((p) => (
                      <UpsertConfigMaps
                        {...p}
                        project={name || ''}
                        editing={record?.metadata?.name}
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
              </Space>
            )
          }
        ]}
      />
    </Card>
  );
};
