import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';

import { queryClient } from '@ui/config/query-client';
import {
  createConfigMap,
  listConfigMaps
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { FieldContainer } from '../../form/field-container';
import { ModalProps } from '../../modal/use-modal';
import { ObjectEditor } from '../../object-editor';

import { confgMapSchema } from './schema';

type Props = ModalProps & {
  systemLevel: boolean;
  project: string;
};

export const CreateConfigMapModal = ({ project, systemLevel, hide, visible }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      name: '',
      data: {}
    },
    resolver: zodResolver(confgMapSchema)
  });

  const { mutate, isPending } = useMutation(createConfigMap, {
    onSuccess: () => {
      queryClient.refetchQueries({
        queryKey: createConnectQueryKey({
          schema: listConfigMaps,
          cardinality: 'finite'
        })
      });
      hide();
    }
  });

  const onSubmit = handleSubmit((data) => mutate({ ...data, project, systemLevel }));

  return (
    <Modal
      okText='Create'
      onOk={onSubmit}
      onCancel={hide}
      open={visible}
      title='Create ConfigMap'
      okButtonProps={{ loading: isPending }}
    >
      <FieldContainer label='Name' name='name' control={control}>
        {({ field }) => <Input {...field} />}
      </FieldContainer>
      <FieldContainer label='Data' name='data' control={control}>
        {({ field }) => <ObjectEditor value={field.value} onChange={field.onChange} />}
      </FieldContainer>
    </Modal>
  );
};
