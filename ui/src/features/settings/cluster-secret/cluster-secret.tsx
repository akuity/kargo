import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Card, Flex, Table, Tag } from 'antd';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  useDeleteSystemGenericCredentials,
  useListSystemGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';
import { useGetConfig } from '@ui/gen/api/v2/system/system';

import { CreateSystemSecretModal } from './create-system-secret-modal';
import { updateSystemSecretCache } from './secret-cache';

export const ClusterSecret = () => {
  const client = useQueryClient();
  const listSystemSecretsQuery = useListSystemGenericCredentials();
  const onSaved = (secret: V1Secret) =>
    updateSystemSecretCache(client, secret.metadata?.name || '', secret);
  const confirm = useConfirmModal();

  const getConfigQuery = useGetConfig();
  const config = getConfigQuery.data?.data;

  const createSecretModal = useModal((p) => <CreateSystemSecretModal {...p} onSuccess={onSaved} />);

  const deleteSecretsMutation = useDeleteSystemGenericCredentials({
    mutation: {
      onSuccess: (_response, variables) =>
        updateSystemSecretCache(client, variables.genericCredentials)
    }
  });

  return (
    <Card
      title={
        <>
          System Secrets{' '}
          <Tag className='text-xs ml-2' color='blue'>
            namespace: {config?.systemResourcesNamespace}
          </Tag>
        </>
      }
      type='inner'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => createSecretModal.show()}>
          Add Secret
        </Button>
      }
    >
      <Table<V1Secret>
        className='my-2'
        scroll={{ x: 'max-content' }}
        dataSource={listSystemSecretsQuery.data?.data?.items || []}
        rowKey={(record) => record?.metadata?.name || ''}
        loading={listSystemSecretsQuery.isLoading}
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
            key: 'secrets',
            render: (_, record) => {
              const secretKeys = Object.keys(record?.stringData || {});

              if (!secretKeys.length) {
                return <Tag color='red'>It looks like this secret is empty.</Tag>;
              }

              return (
                <Flex gap={4} wrap>
                  {secretKeys.map((secretKey) => (
                    <Tag key={secretKey} color='blue'>
                      {secretKey}
                    </Tag>
                  ))}
                </Flex>
              );
            }
          },
          {
            key: 'actions',
            fixed: 'right',
            render: (_, record) => (
              <Flex justify='flex-end' gap={8}>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
                  onClick={() =>
                    createSecretModal.show((p) => (
                      <CreateSystemSecretModal onSuccess={onSaved} init={record} {...p} />
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
                  onClick={() => {
                    confirm({
                      title: 'Delete Secret',
                      content: (
                        <p>Are you sure you want to delete secret {record?.metadata?.name}?</p>
                      ),
                      onOk: () =>
                        deleteSecretsMutation.mutate({
                          genericCredentials: record?.metadata?.name || ''
                        })
                    });
                  }}
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
