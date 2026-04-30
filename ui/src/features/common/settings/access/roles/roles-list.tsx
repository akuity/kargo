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

import { ConfirmModal } from '@ui/features/common/confirm-modal/confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { RbacRole } from '@ui/gen/api/v2/models';
import {
  useDeleteProjectRole,
  useListProjectRoles,
  useListSystemRoles
} from '@ui/gen/api/v2/rbac/rbac';

import { CreateRole } from './create-role';
import { RulesModal } from './rules-modal';

const renderColumn = (key: string) => {
  return {
    title: key.charAt(0).toUpperCase() + key.slice(1),
    key,
    render: (record: RbacRole) => {
      const claimValues = (record.claims || []).find((claim) => claim.name === key)?.values;
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

type Props = {
  project?: string;
  systemLevel?: boolean;
};

export const RolesList = ({ project = '', systemLevel = false }: Props) => {
  const systemRolesQuery = useListSystemRoles({ query: { enabled: systemLevel } });
  const projectRolesQuery = useListProjectRoles(project, {
    query: { enabled: !systemLevel && !!project }
  });
  const activeQuery = systemLevel ? systemRolesQuery : projectRolesQuery;

  const { data, refetch, isLoading } = activeQuery;
  // backend returns json object instead of array of RbacRole for some reason, so we need to cast it
  const roles = (data?.data as unknown as RbacRole[]) || [];

  const [showCreateRole, setShowCreateRole] = useState(false);
  const [editingRole, setEditingRole] = useState<RbacRole | undefined>();

  const { show, hide } = useModal();

  const { mutate: deleteRoleAction } = useDeleteProjectRole({
    mutation: {
      onSuccess: () => {
        hide();
        setTimeout(() => refetch(), 500);
      }
    }
  });

  return (
    <Card
      title='Roles'
      type='inner'
      className='min-h-full'
      extra={
        // System-level roles are read-only
        systemLevel ? null : (
          <Button
            icon={<FontAwesomeIcon icon={faPlus} />}
            onClick={() => {
              setShowCreateRole(true);
              setEditingRole(undefined);
            }}
          >
            Create Role
          </Button>
        )
      }
    >
      {(showCreateRole || editingRole) && (
        <CreateRole
          project={project}
          onSuccess={refetch}
          hide={() => {
            setShowCreateRole(false);
            setEditingRole(undefined);
          }}
          editing={editingRole}
        />
      )}
      <Table
        className='my-2 overflow-x-auto'
        key={roles.length}
        dataSource={roles.sort((a, b) => {
          if (a.metadata?.name && b.metadata?.name) {
            return a.metadata?.name.localeCompare(b.metadata?.name);
          } else {
            return 0;
          }
        })}
        loading={isLoading}
        rowKey={(record: RbacRole) => record?.metadata?.name || ''}
        columns={[
          {
            title: 'Name',
            key: 'name',
            render: (record: RbacRole) => <>{record.metadata?.name}</>
          },
          renderColumn('email'),
          renderColumn('sub'),
          renderColumn('groups'),
          {
            title: 'Rules',
            key: 'rules',
            render: (record: RbacRole) => {
              const rulesCount = record?.rules?.length || 0;
              return (
                <FontAwesomeIcon
                  icon={rulesCount > 0 ? faInfoCircle : faQuestionCircle}
                  className={classNames({
                    'cursor-pointer text-blue-500': rulesCount > 0,
                    'text-gray-200': rulesCount === 0
                  })}
                  onClick={() => {
                    if (rulesCount === 0) return;
                    show((p) => (
                      <RulesModal rules={record.rules || []} name={record?.metadata?.name} {...p} />
                    ));
                  }}
                />
              );
            }
          },
          {
            key: 'actions',
            render: (record: RbacRole) => {
              return (
                <div className='flex items-center justify-end'>
                  {!systemLevel && (
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
                                  project,
                                  role: record.metadata?.name || ''
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
