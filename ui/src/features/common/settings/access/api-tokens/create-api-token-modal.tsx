import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Input, Modal, Select, Typography } from 'antd';
import React from 'react';
import { useForm } from 'react-hook-form';
import z from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { RbacRole } from '@ui/gen/api/v2/models';
import {
  createProjectAPIToken,
  createSystemAPIToken,
  getListProjectAPITokensQueryKey,
  getListSystemAPITokensQueryKey,
  useListProjectRoles,
  useListSystemRoles
} from '@ui/gen/api/v2/rbac/rbac';
import { zodValidators } from '@ui/utils/validators';

type Props = ModalComponentProps & {
  systemLevel: boolean;
  project: string;
};

export const CreateAPITokenModal = ({ hide, visible, systemLevel, project }: Props) => {
  const [token, setToken] = React.useState<string>('');
  const queryClient = useQueryClient();

  const systemRolesQuery = useListSystemRoles({ query: { enabled: systemLevel } });
  const projectRolesQuery = useListProjectRoles(project, {
    query: { enabled: !systemLevel && !!project }
  });
  const listRolesQuery = systemLevel ? systemRolesQuery : projectRolesQuery;
  // backend returns json object instead of array of RbacRole for some reason, so we need to cast it
  const roles = (listRolesQuery.data?.data as unknown as RbacRole[]) || [];

  const { mutate: createAPITokenMutation, isPending } = useMutation({
    mutationFn: ({ name, roleName }: { name: string; roleName: string }) =>
      systemLevel
        ? createSystemAPIToken(roleName, { name })
        : createProjectAPIToken(project, roleName, { name }),
    onSuccess: (response) => {
      const tokenB64 = (response.data?.data as Record<string, string>)?.token;
      setToken(tokenB64 ? atob(tokenB64) : '');
      queryClient.invalidateQueries({
        queryKey: systemLevel
          ? getListSystemAPITokensQueryKey()
          : getListProjectAPITokensQueryKey(project)
      });
    }
  });

  const { control, handleSubmit } = useForm({
    resolver: zodResolver(schema)
  });

  const onSubmit = handleSubmit((data) =>
    createAPITokenMutation({ name: data.name, roleName: data.roleName })
  );

  // On successful creation, show token
  if (token) {
    return (
      <Modal title='API Token Created' onCancel={hide} open={visible} footer={null}>
        <Typography.Paragraph>
          The API Token has been created successfully. Please copy the token below as it will not be
          shown again.
        </Typography.Paragraph>
        <Typography.Paragraph>
          <pre>
            <Typography.Text copyable>{token}</Typography.Text>
          </pre>
        </Typography.Paragraph>
      </Modal>
    );
  }

  return (
    <Modal
      title='Create API Token'
      onCancel={hide}
      open={visible}
      okButtonProps={{ loading: isPending }}
      okText='Create'
      onOk={onSubmit}
    >
      <FieldContainer control={control} label='Name' name='name'>
        {({ field: { value, onChange } }) => (
          <Input placeholder='Token Name' value={value} onChange={onChange} />
        )}
      </FieldContainer>

      <FieldContainer control={control} label='Role' name='roleName'>
        {({ field: { value, onChange } }) => (
          <Select
            value={value}
            onChange={onChange}
            loading={listRolesQuery.isLoading}
            placeholder='Select Role'
            options={roles.map((sa) => ({
              label: sa.metadata?.name || '',
              value: sa.metadata?.name || ''
            }))}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};

const schema = z.object({
  name: zodValidators.requiredString,
  roleName: zodValidators.requiredString
});
