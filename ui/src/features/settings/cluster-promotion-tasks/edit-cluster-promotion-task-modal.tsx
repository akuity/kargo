import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getClusterAnalysisTemplateYAMLExample } from '@ui/features/utils/cluster-analysis-template-example';
import {
  getListClusterPromotionTasksQueryKey,
  useGetClusterPromotionTask
} from '@ui/gen/api/v2/core/core';
import { ClusterPromotionTask } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

type EditClusterPromotionTaskModalProps = ModalComponentProps & {
  clusterPromotionTask: ClusterPromotionTask;
};

export const EditClusterPromotionTaskModal = (props: EditClusterPromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const getClusterPromotionTaskQuery = useGetClusterPromotionTask(
    props.clusterPromotionTask?.metadata?.name || ''
  );

  const updateResourceMutation = useUpdateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListClusterPromotionTasksQueryKey()
        });
        props.hide();
      }
    }
  });

  const taskYAML = getClusterPromotionTaskQuery.data?.data
    ? stringify(getClusterPromotionTaskQuery.data.data)
    : '';

  const editClusterPromotionTaskForm = useForm({
    values: {
      value: taskYAML
    }
  });

  const onSubmit = editClusterPromotionTaskForm.handleSubmit((data) => {
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
      <FieldContainer control={editClusterPromotionTaskForm.control} name='value'>
        {({ field }) => (
          <YamlEditor
            value={field.value}
            onChange={(e) => field.onChange(e || '')}
            isLoading={getClusterPromotionTaskQuery.isFetching}
            height='500px'
            placeholder={getClusterAnalysisTemplateYAMLExample()}
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
