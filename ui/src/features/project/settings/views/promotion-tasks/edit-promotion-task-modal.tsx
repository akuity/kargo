import { createConnectQueryKey, useMutation, useQuery } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  getPromotionTask,
  listPromotionTasks,
  updateResource
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { PromotionTask } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { getPromotionTaskYAMLExample } from './promotion-task-example';

type EditPromotionTaskModalProps = ModalComponentProps & {
  promotionTask: PromotionTask;
};

export const EditPromotionTaskModal = (props: EditPromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const getPromotionTaskQuery = useQuery(getPromotionTask, {
    project: props.promotionTask?.metadata?.namespace,
    name: props.promotionTask?.metadata?.name,
    format: RawFormat.YAML
  });

  const updateResourceMutation = useMutation(updateResource, { onSuccess: () => props.hide() });

  const editPromotionTaskForm = useForm({
    values: {
      value: decodeRawData(getPromotionTaskQuery.data)
    }
  });

  const onSubmit = editPromotionTaskForm.handleSubmit((data) => {
    const textEncoder = new TextEncoder();

    updateResourceMutation.mutate(
      {
        manifest: textEncoder.encode(data.value)
      },
      {
        onSuccess: () =>
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({
              schema: listPromotionTasks,
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
      <FieldContainer control={editPromotionTaskForm.control} name='value'>
        {({ field }) => (
          <YamlEditor
            value={field.value}
            onChange={(e) => field.onChange(e || '')}
            isLoading={getPromotionTaskQuery.isFetching}
            height='500px'
            placeholder={getPromotionTaskYAMLExample(
              props.promotionTask?.metadata?.name || 'custom-promotion-task'
            )}
            isHideManagedFieldsDisplayed
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
