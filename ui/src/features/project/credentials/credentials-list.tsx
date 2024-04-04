import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faCode,
  faExternalLink,
  faPencil,
  faPlus,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Table } from 'antd';
import { useParams } from 'react-router-dom';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
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

  const { data, refetch } = useQuery(listCredentials, { project: name });
  const { mutate } = useMutation(deleteCredentials, {
    onSuccess: () => {
      refetch();
    }
  });

  return (
    <div className='p-4'>
      <h1 className='pl-2 text-lg font-semibold flex items-center mb-4'>
        <Button
          type='primary'
          className='ml-auto'
          icon={<FontAwesomeIcon icon={faPlus} />}
          onClick={() => {
            showCreate((p) => (
              <CreateCredentialsModal project={name || ''} onSuccess={refetch} {...p} />
            ));
          }}
        >
          New
        </Button>
      </h1>
      <Table
        dataSource={data?.credentials || []}
        rowKey={(record) => record?.metadata?.name || ''}
        columns={[
          {
            title: 'Name',
            key: 'name',
            render: (record) => <div>{record?.metadata?.name}</div>
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
                      }
                    });
                  }}
                >
                  Delete
                </Button>
              </div>
            )
          }
        ]}
      />
    </div>
  );
};
