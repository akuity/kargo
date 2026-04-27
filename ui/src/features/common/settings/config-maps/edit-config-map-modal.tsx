import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Checkbox, Input, Modal, Typography } from 'antd';
import { Controller, useForm } from 'react-hook-form';

import { queryClient } from '@ui/config/query-client';
import { REPLICATE_TO_ALL_VALUE, REPLICATE_TO_ANNOTATION_KEY } from '@ui/features/common/utils';
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
      data: { ...(configMap.data || {}) },
      replicate:
        configMap.metadata?.annotations?.[REPLICATE_TO_ANNOTATION_KEY] === REPLICATE_TO_ALL_VALUE
    },
    resolver: zodResolver(confgMapSchema)
  });

  const name = configMap.metadata?.name || '';
  const mutationFn = project
    ? (values: { data: Record<string, string>; replicate?: boolean }) =>
        updateProjectConfigMap(project, name, { data: values.data })
    : (values: { data: Record<string, string>; replicate?: boolean }) =>
        updateSharedConfigMap(name, { data: values.data, replicate: values.replicate });
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
      width={580}
    >
      <FieldContainer label='Name' name='name' control={control}>
        {({ field }) => <Input {...field} disabled />}
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
