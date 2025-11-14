import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import Modal from 'antd/es/modal/Modal';
import Link from 'antd/es/typography/Link';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  createResource,
  listPromotionTasks
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { getPromotionTaskYAMLExample } from './promotion-task-example';

type CreatePromotionTaskModalProps = ModalComponentProps & {
  namespace: string;
};

export const CreatePromotionTaskModal = (props: CreatePromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const createResourceMutation = useMutation(createResource);

  const promotionTaskForm = useForm({
    defaultValues: {
      value: getPromotionTaskYAMLExample(props.namespace)
    }
  });

  const onSubmit = promotionTaskForm.handleSubmit((data) => {
    const textEncoder = new TextEncoder();

    createResourceMutation.mutate(
      {
        manifest: textEncoder.encode(data.value)
      },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({
              schema: listPromotionTasks,
              cardinality: 'finite'
            })
          });
          props.hide();
        }
      }
    );
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
