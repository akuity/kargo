import { useQuery } from '@connectrpc/connect-query';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faIdBadge, faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Table } from 'antd';
import { useParams } from 'react-router-dom';

import { useModal } from '@ui/features/common/modal/use-modal';
import { listCredentials } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { CreateCredentialsModal } from './create-credentials-modal';
import { DeleteCredentialsModal } from './delete-credentials-modal';

export type CredentialsType = 'git' | 'helm' | 'image';
export const CredentialTypeLabelKey = 'kargo.akuity.io/cred-type';

export const IconForCredentialsType = (type: CredentialsType) => {
  switch (type) {
    case 'git':
      return faGit;
    case 'helm':
      return faAnchor;
    case 'image':
      return faDocker;
  }
};

export const CredentialsList = () => {
  const { name } = useParams();
  const { show: showCreate } = useModal();
  const { show: showDelete } = useModal();

  const { data, refetch } = useQuery(listCredentials, { project: name });

  return (
    <div className='p-4'>
      <h1 className='pl-2 text-lg font-semibold flex items-center mb-4'>
        <FontAwesomeIcon icon={faIdBadge} className='mr-2' />
        Credentials
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
                  icon={IconForCredentialsType(
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
              <div>{record?.stringData['repoURL'] || record?.stringData['repoURLPattern']}</div>
            )
          },
          {
            title: 'Username',
            key: 'username',
            render: (record) => <div>{record?.stringData['username']}</div>
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
                    showDelete((p) => (
                      <DeleteCredentialsModal
                        project={name || ''}
                        name={record?.metadata?.name || ''}
                        onSuccess={refetch}
                        {...p}
                      />
                    ));
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
