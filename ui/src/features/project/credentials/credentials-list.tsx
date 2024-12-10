import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faCode,
  faDharmachakra,
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
  listCredentials
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { CreateCredentialsModal } from './create-credentials-modal';
import { CredentialTypeLabelKey, CredentialsDataKey, CredentialsType } from './types';
import { iconForCredentialsType } from './utils';

export const CredentialsList = () => {
  const { name } = useParams();
  const { show: showCreate } = useModal();
  const confirm = useConfirmModal();

  const { data, isLoading, refetch } = useQuery(listCredentials, { project: name });
  const { mutate } = useMutation(deleteCredentials, {
    onSuccess: () => {
      refetch();
    }
  });

  const specificCredentials: Secret[] = [];
  const genericCredentials: Secret[] = [];

  for (const credential of data?.credentials || []) {
    const credentialType = credential?.metadata?.labels?.[
      CredentialTypeLabelKey
    ] as CredentialsType;

    if (credentialType !== 'generic') {
      specificCredentials.push(credential);
      continue;
    }

    genericCredentials.push(credential);
  }

  return (
    <div className='p-4'>
      <Button
        type='primary'
        className='mb-4 text-xs font-semibold'
        icon={<FontAwesomeIcon icon={faPlus} />}
        onClick={() => {
          showCreate((p) => (
            <CreateCredentialsModal project={name || ''} onSuccess={refetch} {...p} />
          ));
        }}
      >
        ADD CREDENTIALS
      </Button>
      <Flex gap={16}>
        <Card title='Repo Credentials' className='w-1/2'>
          <Table
            key={specificCredentials.length}
            dataSource={specificCredentials}
            rowKey={(record: Secret) => record?.metadata?.name || ''}
            loading={isLoading}
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
                            project={name || ''}
                            onSuccess={refetch}
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
                            mutate({ project: name || '', name: record?.metadata?.name || '' });
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
            <>
              <FontAwesomeIcon icon={faDharmachakra} className='mr-2' />
              Generic Kubernetes Secrets
            </>
          }
          className='w-1/2'
        >
          <Table
            key={genericCredentials.length}
            dataSource={genericCredentials}
            rowKey={(record: Secret) => record?.metadata?.name || ''}
            columns={[
              {
                title: 'Secret Resource Name',
                key: 'name',
                render: (record) => record?.metadata?.name
              },
              {
                title: 'Available Secrets',
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
                            project={name || ''}
                            onSuccess={refetch}
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
                            mutate({ project: name || '', name: record?.metadata?.name || '' });
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
            loading={isLoading}
          />
        </Card>
      </Flex>
    </div>
  );
};
