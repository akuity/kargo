import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getListPromotionTasksQueryKey, useGetPromotionTask } from '@ui/gen/api/v2/core/core';
import { PromotionTask } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

import { getPromotionTaskYAMLExample } from './promotion-task-example';

type EditPromotionTaskModalProps = ModalComponentProps & {
  promotionTask: PromotionTask;
};

export const EditPromotionTaskModal = (props: EditPromotionTaskModalProps) => {
  const queryClient = useQueryClient();
  const project = props.promotionTask?.metadata?.namespace || '';
  const taskName = props.promotionTask?.metadata?.name || '';

  const getPromotionTaskQuery = useGetPromotionTask(project, taskName);

  const updateResourceMutation = useUpdateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListPromotionTasksQueryKey(project)
        });
        props.hide();
      }
    }
  });

  const editPromotionTaskForm = useForm({
    values: {
      value: getPromotionTaskQuery.data?.data ? stringify(getPromotionTaskQuery.data.data) : ''
    }
  });

  const onSubmit = editPromotionTaskForm.handleSubmit((data) => {
    updateResourceMutation.mutate({ data: data.value });
  });

  return (
    <Modal
      open={props.visible}
      onCancel={props.hide}
      title='Edit Promotion Task'
      okText='Update'
      onOk={onSubmit}
      okButtonProps={{ loading: updateResourceMutation.isPending }}
      width={700}
    >
      <FieldContainer control={editPromotionTaskForm.control} name='value'>
        {({ field }) => (
          <YamlEditor
            value={field.value}
            onChange={(e) => field.onChange(e || '')}
            isLoading={getPromotionTaskQuery.isFetching}
            height='500px'
            placeholder={getPromotionTaskYAMLExample(taskName || 'custom-promotion-task')}
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
