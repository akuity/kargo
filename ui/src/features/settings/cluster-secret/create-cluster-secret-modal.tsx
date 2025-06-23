import { useMutation } from '@connectrpc/connect-query';
import { faAsterisk } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { dnsRegex } from '@ui/features/common/utils';
import { SecretEditor } from '@ui/features/project/settings/views/credentials/secret-editor';
import {
  createClusterSecret,
  updateClusterSecret
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';
import { zodValidators } from '@ui/utils/validators';

const createFormSchema = z.object({
  name: zodValidators.requiredString.regex(dnsRegex, 'Secret name must be a valid DNS subdomain'),
  data: z.array(z.array(z.string()))
});

type CreateClusterSecretModalProps = ModalComponentProps & {
  init?: Secret;
  onSuccess?(): void;
};

export const CreateClusterSecretModal = (props: CreateClusterSecretModalProps) => {
  const createClusterSecretForm = useForm({
    defaultValues: {
      name: props.init?.metadata?.name || '',
      data: Object.entries(props.init?.stringData || {})
    },
    resolver: zodResolver(createFormSchema)
  });

  const editing = !!props.init;

  const createClusterSecretsMutation = useMutation(createClusterSecret, {
    onSuccess: () => {
      props.hide();
      props.onSuccess?.();
    }
  });

  const updateClusterSecretMutation = useMutation(updateClusterSecret, {
    onSuccess: () => {
      props.hide();
      props.onSuccess?.();
    }
  });

  const onSubmit = createClusterSecretForm.handleSubmit((values) => {
    const data: Record<string, string> = {};

    if (values?.data?.length > 0) {
      for (const [k, v] of values.data) {
        data[k] = v;
      }
    }

    if (editing) {
      return updateClusterSecretMutation.mutate({
        ...values,
        data
      });
    }

    return createClusterSecretsMutation.mutate({
      ...values,
      data
    });
  });

  return (
    <Modal
      open={props.visible}
      onCancel={props.hide}
      okText={editing ? 'Update' : 'Create'}
      title={
        <>
          <FontAwesomeIcon icon={faAsterisk} className='mr-2' />
          {editing ? 'Edit' : 'Create'} Secret
        </>
      }
      onOk={onSubmit}
      width='612px'
    >
      <FieldContainer control={createClusterSecretForm.control} name='name' label='Name'>
        {({ field }) => (
          <Input
            value={field.value as string}
            placeholder='secret-1'
            onChange={(e) => field.onChange(e.target.value)}
          />
        )}
      </FieldContainer>
      <FieldContainer control={createClusterSecretForm.control} name='data' label='Data'>
        {({ field }) => (
          <SecretEditor secret={field.value as [string, string][]} onChange={field.onChange} />
        )}
      </FieldContainer>
    </Modal>
  );
};
