import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faQuestionCircle, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Popover, Space, Table, Tag, Typography } from 'antd';
import React from 'react';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteGenericCredentials,
  listGenericCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { CreateCredentialsModal } from './create-credentials-modal';

type Props = {
  // empty means shared
  project?: string;
  description?: React.ReactNode;
};

export const GenericCredentialsList = ({ project = '', description }: Props) => {
  const confirm = useConfirmModal();

  const listSecretsQuery = useQuery(listGenericCredentials, { project });

  const deleteSecretsMutation = useMutation(deleteGenericCredentials, {
    onSuccess: () => listSecretsQuery.refetch()
  });

  const genericCredentials: Secret[] = listSecretsQuery.data?.credentials || [];

  const { show: showCreateGeneric } = useModal((p) => (
    <CreateCredentialsModal
      type='generic'
      project={project}
      onSuccess={listSecretsQuery.refetch}
      {...p}
    />
  ));

  return (
    <Card
      className='flex-1'
      type='inner'
      title={
        <Space size={4}>
          Generic Project Secrets
          {description && (
            <Popover content={description}>
              <Typography.Text type='secondary'>
                <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
              </Typography.Text>
            </Popover>
          )}
        </Space>
      }
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => showCreateGeneric()}>
          Add Secret
        </Button>
      }
    >
      <Table
        className='my-2'
        scroll={{ x: 'max-content' }}
        key={genericCredentials.length}
        dataSource={genericCredentials}
        rowKey={(record: Secret) => record?.metadata?.name || ''}
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
              const secretsKeys = Object.keys(record?.stringData) || [];

              if (!secretsKeys.length) {
                return <Tag color='red'>It looks like this secret is empty.</Tag>;
              }

              return secretsKeys.map((secretKey) => (
                <Tag key={secretKey} color='blue'>
                  {secretKey}
                </Tag>
              ));
            }
          },
          {
            key: 'actions',
            fixed: 'right',
            render: (record) => (
              <Space>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
                  onClick={() => {
                    showCreateGeneric((p) => (
                      <CreateCredentialsModal
                        type='generic'
                        project={project}
                        onSuccess={listSecretsQuery.refetch}
                        editing
                        init={record}
                        {...p}
                      />
                    ));
                  }}
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
                      title: (
                        <div className='flex items-center'>
                          <FontAwesomeIcon icon={faTrash} className='mr-2' />
                          Delete Credentials
                        </div>
                      ),
                      content: (
                        <p>
                          Are you sure you want to delete credentials{' '}
                          <b>{record?.metadata?.name}</b>?
                        </p>
                      ),
                      onOk: () => {
                        deleteSecretsMutation.mutate({
                          project,
                          name: record?.metadata?.name || ''
                        });
                      },
                      hide: () => {}
                    });
                  }}
                >
                  Delete
                </Button>
              </Space>
            )
          }
        ]}
        expandable={descriptionExpandable()}
        loading={listSecretsQuery.isLoading}
      />
    </Card>
  );
};
