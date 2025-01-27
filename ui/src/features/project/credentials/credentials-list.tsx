import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faCode,
  faExternalLink,
  faPencil,
  faPlus,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table, Tag } from 'antd';
import { useParams } from 'react-router-dom';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';
import {
  deleteCredentials,
  deleteProjectSecret,
  listCredentials,
  listProjectSecrets
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { CreateCredentialsModal } from './create-credentials-modal';
import { CredentialTypeLabelKey, CredentialsDataKey, CredentialsType } from './types';
import { iconForCredentialsType } from './utils';

export const CredentialsList = () => {
  const { name } = useParams();
  const { show: showCreate } = useModal();
  const confirm = useConfirmModal();

  const listCredentialsQuery = useQuery(listCredentials, { project: name });

  const listSecretsQuery = useQuery(listProjectSecrets, { project: name });

  const deleteCredentialsMutation = useMutation(deleteCredentials, {
    onSuccess: () => {
      listCredentialsQuery.refetch();
    }
  });

  const deleteSecretsMutation = useMutation(deleteProjectSecret, {
    onSuccess: () => listSecretsQuery.refetch()
  });

  const specificCredentials: Secret[] = listCredentialsQuery.data?.credentials || [];
  const genericCredentials: Secret[] = listSecretsQuery.data?.secrets || [];

  return (
    <div className='p-4'>
      <Flex gap={16}>
        <Card
          title={
            <Flex align='center'>
              Repo Credentials
              <Button
                type='primary'
                className='ml-auto text-xs font-semibold'
                icon={<FontAwesomeIcon icon={faPlus} />}
                onClick={() => {
                  showCreate((p) => (
                    <CreateCredentialsModal
                      type='repo'
                      project={name || ''}
                      onSuccess={listCredentialsQuery.refetch}
                      {...p}
                    />
                  ));
                }}
              >
                ADD CREDENTIALS
              </Button>
            </Flex>
          }
          className='w-1/2'
        >
          <Table
            scroll={{ x: 'max-content' }}
            key={specificCredentials.length}
            dataSource={specificCredentials}
            rowKey={(record: Secret) => record?.metadata?.name || ''}
            loading={listCredentialsQuery.isLoading}
            columns={[
              {
                title: 'Name',
                key: 'name',
                render: (record) => {
                  return <div>{record?.metadata?.name}</div>;
                }
              },
              {
                title: 'Type',
                key: 'type',
                render: (record) => (
                  <div className='flex items-center font-semibold text-sm'>
                    <FontAwesomeIcon
                      icon={iconForCredentialsType(
                        record?.metadata?.labels[CredentialTypeLabelKey] as CredentialsType
                      )}
                      className='mr-3 text-blue-500'
                    />
                    {record?.metadata?.labels[CredentialTypeLabelKey].toUpperCase()}
                  </div>
                )
              },
              {
                title: 'Repo URL / Pattern',
                key: 'createdAt',
                render: (record) => (
                  <div className='flex items-center'>
                    <FontAwesomeIcon
                      icon={
                        record.stringData[CredentialsDataKey.RepoUrlIsRegex] === 'true'
                          ? faCode
                          : faExternalLink
                      }
                      className='mr-2'
                    />
                    {record?.stringData[CredentialsDataKey.RepoUrl]}
                  </div>
                )
              },
              {
                title: 'Username',
                key: 'username',
                render: (record) => <div>{record?.stringData[CredentialsDataKey.Username]}</div>
              },
              {
                key: 'actions',
                render: (record) => (
                  <div className='flex items-center w-full'>
                    <Button
                      icon={<FontAwesomeIcon icon={faPencil} />}
                      className='mr-2 ml-auto'
                      onClick={() => {
                        showCreate((p) => (
                          <CreateCredentialsModal
                            type='repo'
                            project={name || ''}
                            onSuccess={listCredentialsQuery.refetch}
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
                      icon={<FontAwesomeIcon icon={faTrash} />}
                      danger
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
                            deleteCredentialsMutation.mutate({
                              project: name || '',
                              name: record?.metadata?.name || ''
                            });
                          },
                          hide: () => {}
                        });
                      }}
                    >
                      Delete
                    </Button>
                  </div>
                )
              }
            ]}
            expandable={descriptionExpandable()}
          />
        </Card>

        <Card
          title={
            <Flex align='center'>
              Generic Project Secrets
              <Button
                type='primary'
                className='ml-auto text-xs font-semibold'
                icon={<FontAwesomeIcon icon={faPlus} />}
                onClick={() => {
                  showCreate((p) => (
                    <CreateCredentialsModal
                      type='generic'
                      project={name || ''}
                      onSuccess={listSecretsQuery.refetch}
                      {...p}
                    />
                  ));
                }}
              >
                ADD SECRET
              </Button>
            </Flex>
          }
          className='w-1/2'
        >
          <Table
            scroll={{ x: 'max-content' }}
            key={genericCredentials.length}
            dataSource={genericCredentials}
            rowKey={(record: Secret) => record?.metadata?.name || ''}
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
                render: (record) => (
                  <div className='flex items-center w-full'>
                    <Button
                      icon={<FontAwesomeIcon icon={faPencil} />}
                      className='mr-2 ml-auto'
                      onClick={() => {
                        showCreate((p) => (
                          <CreateCredentialsModal
                            type='generic'
                            project={name || ''}
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
                      icon={<FontAwesomeIcon icon={faTrash} />}
                      danger
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
                              project: name || '',
                              name: record?.metadata?.name || ''
                            });
                          },
                          hide: () => {}
                        });
                      }}
                    >
                      Delete
                    </Button>
                  </div>
                )
              }
            ]}
            expandable={descriptionExpandable()}
            loading={listSecretsQuery.isLoading}
          />
        </Card>
      </Flex>
    </div>
  );
};
