import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
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

export const ClusterSecret = () => {
  const listSystemSecretsQuery = useListSystemGenericCredentials();
  const confirm = useConfirmModal();

  const getConfigQuery = useGetConfig();
  const config = getConfigQuery.data?.data;

  const createSecretModal = useModal((p) => (
    <CreateSystemSecretModal {...p} onSuccess={listSystemSecretsQuery.refetch} />
  ));

  const deleteSecretsMutation = useDeleteSystemGenericCredentials({
    mutation: {
      onSuccess: () => listSystemSecretsQuery.refetch()
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

              return secretKeys.map((secretKey) => (
                <Tag key={secretKey} color='blue'>
                  {secretKey}
                </Tag>
              ));
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
                      <CreateSystemSecretModal
                        onSuccess={listSystemSecretsQuery.refetch}
                        init={record}
                        {...p}
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
