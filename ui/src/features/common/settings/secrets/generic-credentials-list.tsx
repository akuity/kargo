import { faPencil, faPlus, faQuestionCircle, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Button, Card, Popover, Space, Table, Tag, Typography } from 'antd';
import React from 'react';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteProjectGenericCredentials,
  deleteSharedGenericCredentials,
  useListProjectGenericCredentials,
  useListSharedGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';

import { CreateCredentialsModal } from './create-credentials-modal';

type Props = {
  // empty means shared
  project?: string;
  description?: React.ReactNode;
};

export const GenericCredentialsList = ({ project = '', description }: Props) => {
  const confirm = useConfirmModal();

  const sharedQuery = useListSharedGenericCredentials({ query: { enabled: !project } });
  const projectQuery = useListProjectGenericCredentials(project, { query: { enabled: !!project } });
  const listSecretsQuery = project ? projectQuery : sharedQuery;

  const deleteSecretsMutation = useMutation({
    mutationFn: (name: string) =>
      project
        ? deleteProjectGenericCredentials(project, name)
        : deleteSharedGenericCredentials(name),
    onSuccess: () => listSecretsQuery.refetch()
  });

  const genericCredentials: V1Secret[] = listSecretsQuery.data?.data?.items || [];

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
      <Table<V1Secret>
        className='my-2'
        scroll={{ x: 'max-content' }}
        key={genericCredentials.length}
        dataSource={genericCredentials}
        rowKey={(record) => record?.metadata?.name || ''}
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
            key: 'secrets',
            render: (_, record) => {
              const secretsKeys = Object.keys(record?.stringData || {});

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
            render: (_, record) => {
              if (record?.metadata?.labels?.['kargo.akuity.io/replicated-from']) {
                return (
                  <Typography.Text type='secondary' italic>
                    Replicated
                  </Typography.Text>
                );
              }
              return (
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
                          deleteSecretsMutation.mutate(record?.metadata?.name || '');
                        },
                        hide: () => {}
                      });
                    }}
                  >
                    Delete
                  </Button>
                </Space>
              );
            }
          }
        ]}
        expandable={descriptionExpandable()}
        loading={listSecretsQuery.isLoading}
      />
    </Card>
  );
};
