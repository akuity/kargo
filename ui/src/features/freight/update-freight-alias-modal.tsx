import { useMutation } from '@connectrpc/connect-query';
import { faPencil } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal, message } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getAlias } from '@ui/features/common/utils';
import { updateFreightAlias } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { zodValidators } from '@ui/utils/validators';

type Props = ModalComponentProps & {
  freight: Freight;
  project: string;
  onSubmit: (newAlias: string) => void;
};

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const UpdateFreightAliasModal = ({ freight, project, onSubmit, hide, ...props }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: ''
    },
    resolver: zodResolver(formSchema)
  });

  const { mutateAsync: updateAliasAction } = useMutation(updateFreightAlias, {
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success('Alias successfully updated');
    }
  });

  return (
    <Modal
      {...props}
      title={
        <>
          <FontAwesomeIcon icon={faPencil} className='mr-2' />
          Update Alias
        </>
      }
      onCancel={hide}
      okText='Submit'
      onOk={handleSubmit(async (data) => {
        await updateAliasAction({
          project,
          name: freight?.metadata?.name || '',
          newAlias: data.value || ''
        });
        onSubmit(data.value || '');
      })}
    >
      <div className='mb-4'>
        <div className='text-xs font-semibold uppercase'>Freight ID</div>
        <div className='font-mono'>{freight?.metadata?.name}</div>
      </div>
      <FieldContainer label='New Alias' name='value' control={control}>
        {({ field }) => (
          <Input {...field} type='text' placeholder={getAlias(freight) || 'New alias'} />
        )}
      </FieldContainer>
    </Modal>
  );
};
