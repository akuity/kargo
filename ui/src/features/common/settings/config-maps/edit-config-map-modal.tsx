import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';

import { queryClient } from '@ui/config/query-client';
import {
  getListProjectConfigMapsQueryKey,
  getListSharedConfigMapsQueryKey,
  updateProjectConfigMap,
  updateSharedConfigMap
} from '@ui/gen/api/v2/core/core';
import { V1ConfigMap } from '@ui/gen/api/v2/models';

import { FieldContainer } from '../../form/field-container';
import { ModalProps } from '../../modal/use-modal';
import { ObjectEditor } from '../../object-editor';

import { confgMapSchema } from './schema';

type Props = ModalProps & {
  project: string;
  configMap: V1ConfigMap;
};

export const EditConfigMapModal = ({ configMap, project, hide, visible }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      name: configMap.metadata?.name,
      data: { ...(configMap.data || {}) }
    },
    resolver: zodResolver(confgMapSchema)
  });

  const name = configMap.metadata?.name || '';
  const mutationFn = project
    ? (values: { data: Record<string, string> }) =>
        updateProjectConfigMap(project, name, { data: values.data })
    : (values: { data: Record<string, string> }) =>
        updateSharedConfigMap(name, { data: values.data });
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
      okText='Update'
      onOk={onSubmit}
      onCancel={hide}
      open={visible}
      title='Edit ConfigMap'
      okButtonProps={{ loading: isPending }}
    >
      <FieldContainer label='Name' name='name' control={control}>
        {({ field }) => <Input {...field} disabled />}
      </FieldContainer>
      <FieldContainer label='Data' name='data' control={control}>
        {({ field }) => <ObjectEditor value={field.value} onChange={field.onChange} />}
      </FieldContainer>
    </Modal>
  );
};
