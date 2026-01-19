import { useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SecretEditor } from '@ui/features/common/settings/secrets/secret-editor';
import { dnsRegex } from '@ui/features/common/utils';
import {
  createGenericCredentials,
  updateGenericCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';
import { zodValidators } from '@ui/utils/validators';

const createFormSchema = z.object({
  name: zodValidators.requiredString.regex(dnsRegex, 'Secret name must be a valid DNS subdomain'),
  data: z.array(z.array(z.string()))
});

type CreateSystemSecretModalProps = ModalComponentProps & {
  init?: Secret;
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

  const createSystemSecretsMutation = useMutation(createGenericCredentials, {
    onSuccess: () => {
      props.hide();
      props.onSuccess?.();
    }
  });

  const updateSystemSecretMutation = useMutation(updateGenericCredentials, {
    onSuccess: () => {
      props.hide();
      props.onSuccess?.();
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
        systemLevel: true,
        ...values,
        data
      });
    }

    return createSystemSecretsMutation.mutate({
      systemLevel: true,
      ...values,
      data
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
