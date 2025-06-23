import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table, Tag } from 'antd';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { useConfigContext } from '@ui/features/config/use-config-context';
import {
  deleteClusterSecret,
  listClusterSecrets
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { CreateClusterSecretModal } from './create-cluster-secret-modal';

export const ClusterSecret = () => {
  const listClusterSecretsQuery = useQuery(listClusterSecrets);
  const confirm = useConfirmModal();
  const { config } = useConfigContext();

  const createSecretModal = useModal((p) => (
    <CreateClusterSecretModal {...p} onSuccess={listClusterSecretsQuery.refetch} />
  ));

  const deleteSecretsMutation = useMutation(deleteClusterSecret, {
    onSuccess: () => listClusterSecretsQuery.refetch()
  });

  return (
    <Card
      title={
        <>
          Cluster Secret{' '}
          <Tag className='text-xs ml-2' color='blue'>
            namespace: {config?.clusterSecretsNamespace}
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
      <Table
        className='my-2'
        scroll={{ x: 'max-content' }}
        dataSource={listClusterSecretsQuery.data?.secrets || []}
        rowKey={(record) => record?.metadata?.name || ''}
        loading={listClusterSecretsQuery.isLoading}
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
        size='small'
        columns={[
          {
            title: 'Name',
            key: 'name',
            render: (record) => record?.metadata?.name
          },
          {
            title: 'Keys',
            key: 'secrets',
            render: (_, record) => {
              const secretKeys = Object.keys(record?.stringData) || [];

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
            render: (_, record) => (
              <Flex>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
                  className='ml-auto'
                  onClick={() =>
                    createSecretModal.show((p) => (
                      <CreateClusterSecretModal
                        onSuccess={listClusterSecretsQuery.refetch}
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
                  className='ml-2'
                  onClick={() => {
                    confirm({
                      title: (
                        <div className='flex items-center'>
                          <FontAwesomeIcon icon={faTrash} className='mr-2' />
                          Delete Secret
                        </div>
                      ),
                      content: (
                        <p>Are you sure you want to delete secret {record?.metadata?.name}?</p>
                      ),
                      onOk: () =>
                        deleteSecretsMutation.mutate({
                          name: record?.metadata?.name
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
