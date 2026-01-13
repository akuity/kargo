import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faCode,
  faExternalLink,
  faPencil,
  faPlus,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Space, Table } from 'antd';
import Card from 'antd/es/card/Card';
import { useParams } from 'react-router-dom';

import {
  deleteRepoCredentials,
  listRepoCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { useConfirmModal } from '../../confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '../../description-expandable';
import { useModal } from '../../modal/use-modal';

import { CreateCredentialsModal } from './create-credentials-modal';
import { CredentialsDataKey, CredentialsType, CredentialTypeLabelKey } from './types';
import { iconForCredentialsType } from './utils';

export const CredentialsList = () => {
  const { name = '' } = useParams();
  const confirm = useConfirmModal();

  const listCredentialsQuery = useQuery(listRepoCredentials, { project: name });

  const { show: showCreate } = useModal((p) => (
    <CreateCredentialsModal
      type='repo'
      project={name || ''}
      onSuccess={listCredentialsQuery.refetch}
      {...p}
    />
  ));
  const deleteCredentialsMutation = useMutation(deleteRepoCredentials, {
    onSuccess: () => {
      listCredentialsQuery.refetch();
    }
  });

  const specificCredentials: Secret[] = listCredentialsQuery.data?.credentials || [];

  return (
    <Card
      className='flex-1'
      title='Repo Credentials'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => showCreate()}>
          Add Credentials
        </Button>
      }
      type='inner'
    >
      <Table
        className='my-2'
        scroll={{ x: 'max-content' }}
        key={specificCredentials.length}
        dataSource={specificCredentials}
        rowKey={(record: Secret) => record?.metadata?.name || ''}
        loading={listCredentialsQuery.isLoading}
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
        size='small'
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
            fixed: 'right',
            render: (record) => (
              <Space>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  color='default'
                  variant='filled'
                  size='small'
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
              </Space>
            )
          }
        ]}
        expandable={descriptionExpandable()}
      />
    </Card>
  );
};
