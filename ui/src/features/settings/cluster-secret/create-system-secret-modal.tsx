import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SecretEditor } from '@ui/features/common/settings/secrets/secret-editor';
import { dnsRegex } from '@ui/features/common/utils';
import {
  useCreateSystemGenericCredentials,
  useUpdateSystemGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';
import { zodValidators } from '@ui/utils/validators';

const createFormSchema = z.object({
  name: zodValidators.requiredString.regex(dnsRegex, 'Secret name must be a valid DNS subdomain'),
  data: z.array(z.array(z.string()))
});

type CreateSystemSecretModalProps = ModalComponentProps & {
  init?: V1Secret;
  onSuccess?(): void;
};

export const CreateSystemSecretModal = (props: CreateSystemSecretModalProps) => {
  const createSystemSecretForm = useForm({
    defaultValues: {
      name: props.init?.metadata?.name || '',
      data: Object.entries(props.init?.stringData || {})
    },
    resolver: zodResolver(createFormSchema)
  });

  const editing = !!props.init;

  const createSystemSecretsMutation = useCreateSystemGenericCredentials({
    mutation: {
      onSuccess: () => {
        props.hide();
        props.onSuccess?.();
      }
    }
  });

  const updateSystemSecretMutation = useUpdateSystemGenericCredentials({
    mutation: {
      onSuccess: () => {
        props.hide();
        props.onSuccess?.();
      }
    }
  });

  const onSubmit = createSystemSecretForm.handleSubmit((values) => {
    const data: Record<string, string> = {};

    if (values?.data?.length > 0) {
      for (const [k, v] of values.data) {
        data[k] = v;
      }
    }

    if (editing) {
      return updateSystemSecretMutation.mutate({
        genericCredentials: values.name,
        data: { data }
      });
    }

    return createSystemSecretsMutation.mutate({
      data: { name: values.name, data }
    });
  });

  return (
    <Modal
      open={props.visible}
      onCancel={props.hide}
      okText={editing ? 'Update' : 'Create'}
      title={`${editing ? 'Edit' : 'Create'} Secret`}
      onOk={onSubmit}
      width='612px'
    >
      <FieldContainer control={createSystemSecretForm.control} name='name' label='Name'>
        {({ field }) => (
          <Input
            value={field.value as string}
            placeholder='secret-1'
            onChange={(e) => field.onChange(e.target.value)}
            disabled={editing}
          />
        )}
      </FieldContainer>
      <FieldContainer control={createSystemSecretForm.control} name='data' label='Data'>
        {({ field }) => (
          <SecretEditor secret={field.value as [string, string][]} onChange={field.onChange} />
        )}
      </FieldContainer>
    </Modal>
  );
};
