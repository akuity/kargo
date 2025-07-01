import { createConnectQueryKey, useMutation, useQuery } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getClusterAnalysisTemplateYAMLExample } from '@ui/features/utils/cluster-analysis-template-example';
import {
  getClusterPromotionTask,
  listClusterPromotionTasks,
  updateResource
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { ClusterPromotionTask } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

type EditClusterPromotionTaskModalProps = ModalComponentProps & {
  clusterPromotionTask: ClusterPromotionTask;
};

export const EditClusterPromotionTaskModal = (props: EditClusterPromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const getClusterPromotionTaskQuery = useQuery(getClusterPromotionTask, {
    name: props.clusterPromotionTask?.metadata?.name,
    format: RawFormat.YAML
  });

  const updateResourceMutation = useMutation(updateResource, { onSuccess: () => props.hide() });

  const editClusterPromotionTaskForm = useForm({
    values: {
      value: decodeRawData(getClusterPromotionTaskQuery.data)
    }
  });

  const onSubmit = editClusterPromotionTaskForm.handleSubmit((data) => {
    const textEncoder = new TextEncoder();

    updateResourceMutation.mutate(
      {
        manifest: textEncoder.encode(data.value)
      },
      {
        onSuccess: () =>
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({
              schema: listClusterPromotionTasks,
              cardinality: 'infinite'
            })
          })
      }
    );
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
            isHideManagedFieldsDisplayed
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
