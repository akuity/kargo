import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faInfoCircle,
  faPencil,
  faPlus,
  faQuestionCircle,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Space, Table } from 'antd';
import classNames from 'classnames';
import { useState } from 'react';
import { useParams } from 'react-router-dom';

import { ConfirmModal } from '@ui/features/common/confirm-modal/confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { Role } from '@ui/gen/api/rbac/v1alpha1/generated_pb';
import {
  deleteRole,
  listRoles
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { CreateRole } from './create-role';
import { RulesModal } from './rules-modal';

const renderColumn = (key: string) => {
  return {
    title: key.charAt(0).toUpperCase() + key.slice(1),
    key,
    render: (record: Role) => {
      const claimValues = record.claims.find((claim) => claim.name === key)?.values;
      return (
        <div>
          {((claimValues as string[]) || []).length > 0 ? (
            claimValues?.join(',')
          ) : (
            <FontAwesomeIcon icon={faQuestionCircle} className='text-gray-200' />
          )}
        </div>
      );
    }
  };
};

export const RolesSettings = () => {
  const { name } = useParams();
  const { data, refetch } = useQuery(listRoles, { project: name });

  const [showCreateRole, setShowCreateRole] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | undefined>();

  const { show, hide } = useModal();

  const { mutate: deleteRoleAction } = useMutation(deleteRole, {
    onSuccess: () => {
      hide();
      setTimeout(() => refetch(), 500);
    }
  });

  return (
    <Card
      title='Roles'
      type='inner'
      className='min-h-full'
      extra={
        <Button
          icon={<FontAwesomeIcon icon={faPlus} />}
          onClick={() => {
            setShowCreateRole(true);
            setEditingRole(undefined);
          }}
        >
          Create Role
        </Button>
      }
    >
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
        className='my-2'
        key={data?.roles?.length}
        dataSource={(data?.roles || []).sort((a, b) => {
          if (a.metadata?.name && b.metadata?.name) {
            return a.metadata?.name.localeCompare(b.metadata?.name);
          } else {
            return 0;
          }
        })}
        rowKey={(record: Role) => record?.metadata?.name || ''}
        columns={[
          {
            title: 'Name',
            key: 'name',
            render: (record: Role) => <>{record.metadata?.name}</>
          },
          renderColumn('email'),
          renderColumn('sub'),
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
                    show((p) => (
                      <RulesModal rules={record.rules} name={record?.metadata?.name} {...p} />
                    ));
                  }}
                />
              );
            }
          },
          {
            key: 'actions',
            render: (record: Role) => {
              return (
                <div className='flex items-center justify-end'>
                  {record?.kargoManaged && (
                    <Space>
                      <Button
                        icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                        color='default'
                        variant='filled'
                        size='small'
                        onClick={() => {
                          setEditingRole(record);
                          setShowCreateRole(false);
                        }}
                      >
                        Edit
                      </Button>
                      <Button
                        color='danger'
                        variant='filled'
                        size='small'
                        icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                        onClick={() => {
                          show((p) => (
                            <ConfirmModal
                              title='Delete Role'
                              content='Are you sure you want to delete this role?'
                              okButtonProps={{ danger: true }}
                              okText='Yes, Delete'
                              onOk={() => {
                                deleteRoleAction({
                                  name: record.metadata?.name || '',
                                  project: name
                                });
                                refetch();
                              }}
                              {...p}
                            />
                          ));
                        }}
                      >
                        Delete
                      </Button>
                    </Space>
                  )}
                </div>
              );
            }
          }
        ]}
        expandable={descriptionExpandable()}
        pagination={{ hideOnSinglePage: true }}
      />
    </Card>
  );
};
