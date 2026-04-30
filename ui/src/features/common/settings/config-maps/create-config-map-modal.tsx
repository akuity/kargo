import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Checkbox, Input, Modal, Typography } from 'antd';
import { Controller, useForm } from 'react-hook-form';

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
      data: {} as Record<string, string>,
      replicate: false
    },
    resolver: zodResolver(confgMapSchema)
  });

  const mutationFn = project
    ? (values: { name: string; data: Record<string, string>; replicate?: boolean }) =>
        createProjectConfigMap(project, values)
    : (values: { name: string; data: Record<string, string>; replicate?: boolean }) =>
        createSharedConfigMap(values);
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
      width={580}
    >
      <FieldContainer label='Name' name='name' control={control}>
        {({ field }) => <Input {...field} />}
      </FieldContainer>
      {!project && (
        <Controller
          name='replicate'
          control={control}
          render={({ field }) => (
            <label className='flex items-start gap-2 cursor-pointer mb-4'>
              <Checkbox checked={field.value} onChange={(e) => field.onChange(e.target.checked)} />
              <div>
                <div>Replicate</div>
                <Typography.Text type='secondary'>
                  Replicate the resource to all projects to be used by AnalysisTemplates
                </Typography.Text>
              </div>
            </label>
          )}
        />
      )}
      <FieldContainer label='Data' name='data' control={control}>
        {({ field }) => <ObjectEditor value={field.value} onChange={field.onChange} />}
      </FieldContainer>
    </Modal>
  );
};
