import { createConnectQueryKey, useMutation, useQuery } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal, Select, Typography } from 'antd';
import React from 'react';
import { useForm } from 'react-hook-form';
import z from 'zod';

import { queryClient } from '@ui/config/query-client';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  createAPIToken,
  listAPITokens,
  listRoles
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

type Props = ModalComponentProps & {
  systemLevel: boolean;
  project: string;
};

export const CreateAPITokenModal = ({ hide, visible, systemLevel, project }: Props) => {
  const [token, setToken] = React.useState<string>('');
  const listRolesQuery = useQuery(listRoles, { systemLevel, project });
  const { mutate: createAPITokenMutation, isPending } = useMutation(createAPIToken, {
    onSuccess: (d) => {
      setToken(
        decodeRawData({
          result: {
            case: 'raw',
            value: d.tokenSecret?.data.token || new Uint8Array()
          }
        })
      );
      queryClient.refetchQueries({
        queryKey: createConnectQueryKey({
          schema: listAPITokens,
          cardinality: 'finite'
        })
      });
    }
  });

  const { control, handleSubmit } = useForm({
    resolver: zodResolver(schema)
  });

  const onSubmit = handleSubmit((data) =>
    createAPITokenMutation({
      systemLevel,
      project,
      ...data
    })
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
            options={(listRolesQuery.data?.roles || []).map((sa) => ({
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
