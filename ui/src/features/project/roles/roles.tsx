import { useQuery } from '@connectrpc/connect-query';
import {
  faInfoCircle,
  faPencil,
  faPlus,
  faQuestionCircle,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Table } from 'antd';
import classNames from 'classnames';
import { useState } from 'react';
import { useParams } from 'react-router-dom';

import { useModal } from '@ui/features/common/modal/use-modal';
import { listRoles } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Role } from '@ui/gen/service/v1alpha1/service_pb';

import { CreateRole } from './create-role';
import { RulesModal } from './rules-modal';

const renderColumn = (key: keyof Role) => {
  return {
    title: key.charAt(0).toUpperCase() + key.slice(1),
    key,
    render: (record: Role) => {
      return (
        <div>
          {((record[key] as string[]) || []).length > 0 ? (
            (record[key] as string[]).join(',')
          ) : (
            <FontAwesomeIcon icon={faQuestionCircle} className='text-gray-200' />
          )}
        </div>
      );
    }
  };
};

export const Roles = () => {
  const { name } = useParams();
  const { data, refetch } = useQuery(listRoles, { project: name });

  const [showCreateRole, setShowCreateRole] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | undefined>();

  const { show } = useModal();

  return (
    <div className='p-4'>
      {(showCreateRole || editingRole) && (
        <CreateRole
          project={name || ''}
          onSuccess={refetch}
          hide={() => {
            setShowCreateRole(false);
            setEditingRole(undefined);
          }}
          editing={editingRole}
        />
      )}
      <Table
        dataSource={data?.roles || []}
        rowKey={(record) => record?.name || ''}
        columns={[
          {
            title: 'Name',
            key: 'name',
            dataIndex: 'name'
          },
          renderColumn('emails'),
          renderColumn('subs'),
          renderColumn('groups'),
          {
            title: 'Rules',
            key: 'rules',
            render: (record: Role) => {
              return (
                <FontAwesomeIcon
                  icon={record?.rules?.length > 0 ? faInfoCircle : faQuestionCircle}
                  className={classNames({
                    'cursor-pointer text-blue-500': record?.rules?.length > 0,
                    'text-gray-200': record?.rules?.length === 0
                  })}
                  onClick={() => {
                    if (record?.rules?.length === 0) return;
                    show((p) => <RulesModal rules={record.rules} {...p} />);
                  }}
                />
              );
            }
          },
          {
            title: (
              <div className='w-full flex'>
                <Button
                  type='primary'
                  className='ml-auto text-xs font-semibold'
                  icon={<FontAwesomeIcon icon={faPlus} />}
                  onClick={() => {
                    setShowCreateRole(true);
                    setEditingRole(undefined);
                  }}
                >
                  CREATE ROLE
                </Button>
              </div>
            ),
            key: 'actions',
            render: (record: Role) => {
              return (
                <div className='flex items-center justify-end'>
                  {record?.kargoManaged && (
                    <>
                      <Button
                        icon={<FontAwesomeIcon icon={faPencil} />}
                        className='mr-2'
                        onClick={() => {
                          setEditingRole(record);
                          setShowCreateRole(false);
                        }}
                      >
                        Edit
                      </Button>
                      <Button type='primary' icon={<FontAwesomeIcon icon={faTrash} />} danger>
                        Delete
                      </Button>
                    </>
                  )}
                </div>
              );
            }
          }
        ]}
      />
    </div>
  );
};
