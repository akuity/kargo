import { useQueryClient } from '@tanstack/react-query';
import Modal from 'antd/es/modal/Modal';
import Link from 'antd/es/typography/Link';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getListPromotionTasksQueryKey } from '@ui/gen/api/v2/core/core';
import { useCreateResource } from '@ui/gen/api/v2/resources/resources';

import { getPromotionTaskYAMLExample } from './promotion-task-example';

type CreatePromotionTaskModalProps = ModalComponentProps & {
  namespace: string;
};

export const CreatePromotionTaskModal = (props: CreatePromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const createResourceMutation = useCreateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListPromotionTasksQueryKey(props.namespace)
        });
        props.hide();
      }
    }
  });

  const promotionTaskForm = useForm({
    defaultValues: {
      value: getPromotionTaskYAMLExample(props.namespace)
    }
  });

  const onSubmit = promotionTaskForm.handleSubmit((data) => {
    createResourceMutation.mutate({ data: data.value });
  });

  return (
    <Modal
      open={props.visible}
      onCancel={props.hide}
      title={
        <>
          Create Promotion Task{' '}
          <Link
            href='https://docs.kargo.io/user-guide/reference-docs/promotion-tasks/'
            target='_blank'
            className='ml-1 text-xs'
          >
            documentation
          </Link>
        </>
      }
      okText='Create'
      width={700}
      onOk={onSubmit}
    >
      <FieldContainer control={promotionTaskForm.control} name='value'>
        {({ field }) => (
          <YamlEditor
            value={field.value}
            onChange={(e) => field.onChange(e || '')}
            height='500px'
            placeholder={getPromotionTaskYAMLExample(props.namespace)}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
