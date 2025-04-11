import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Flex, Space } from 'antd';
import Modal from 'antd/es/modal/Modal';
import Link from 'antd/es/typography/Link';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  createResource,
  listClusterPromotionTasks
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { getClusterPromotionTaskYAMLExample } from './cluster-promotion-task-example';

type CreateClusterPromotionTaskModalProps = ModalComponentProps;

export const CreateClusterPromotionTaskModal = (props: CreateClusterPromotionTaskModalProps) => {
  const queryClient = useQueryClient();

  const createResourceMutation = useMutation(createResource);

  const clusterPromotionTaskForm = useForm({
    defaultValues: {
      value: getClusterPromotionTaskYAMLExample()
    }
  });

  const onSubmit = clusterPromotionTaskForm.handleSubmit((data) => {
    const textEncoder = new TextEncoder();

    createResourceMutation.mutate(
      {
        manifest: textEncoder.encode(data.value)
      },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({
              schema: listClusterPromotionTasks,
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
      title='Create Cluster Promotion Task'
      width={700}
      footer={
        <Flex justify='space-between' align='center'>
          <Link
            href='https://docs.kargo.io/user-guide/reference-docs/promotion-tasks/#defining-a-global-promotion-task'
            target='_blank'
          >
            Documentation
          </Link>
          <Space>
            <Button onClick={props.hide}>Cancel</Button>
            <Button onClick={onSubmit} loading={createResourceMutation.isPending} type='primary'>
              Create
            </Button>
          </Space>
        </Flex>
      }
    >
      <FieldContainer control={clusterPromotionTaskForm.control} name='value'>
        {({ field }) => (
          <YamlEditor
            value={field.value}
            onChange={(e) => field.onChange(e || '')}
            height='500px'
            placeholder={getClusterPromotionTaskYAMLExample()}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
