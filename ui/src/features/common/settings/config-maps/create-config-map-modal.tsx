import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';

import { queryClient } from '@ui/config/query-client';
import {
  createProjectConfigMap,
  createSharedConfigMap,
  getListProjectConfigMapsQueryKey,
  getListSharedConfigMapsQueryKey
} from '@ui/gen/api/v2/core/core';

import { FieldContainer } from '../../form/field-container';
import { ModalProps } from '../../modal/use-modal';
import { ObjectEditor } from '../../object-editor';

import { confgMapSchema } from './schema';

type Props = ModalProps & {
  project: string;
};

export const CreateConfigMapModal = ({ project, hide, visible }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      name: '',
      data: {} as Record<string, string>
    },
    resolver: zodResolver(confgMapSchema)
  });

  const mutationFn = project
    ? (values: { name: string; data: Record<string, string> }) =>
        createProjectConfigMap(project, values)
    : (values: { name: string; data: Record<string, string> }) => createSharedConfigMap(values);
  const queryKey = project
    ? getListProjectConfigMapsQueryKey(project)
    : getListSharedConfigMapsQueryKey();

  const { mutate, isPending } = useMutation({
    mutationFn,
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey });
      hide();
    }
  });

  const onSubmit = handleSubmit((data) => mutate(data));

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
