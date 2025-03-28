import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table } from 'antd';
import { format } from 'date-fns';
import { useParams } from 'react-router-dom';

import {
  deleteResource,
  listPromotionTasks
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { PromotionTask } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useConfirmModal } from '../common/confirm-modal/use-confirm-modal';
import { useModal } from '../common/modal/use-modal';
import { PromotionTaskManifestsGen } from '../utils/manifest-generator';

import { CreatePromotionTaskModal } from './create-promotion-task';
import { EditPromotionTaskModal } from './edit-promotion-task-modal';

export const PromotionTasks = () => {
  const { name } = useParams();
  const confirm = useConfirmModal();

  const listPromotionTasksQuery = useQuery(listPromotionTasks, { project: name });

  const deletePromotionTaskMutation = useMutation(deleteResource);

  const promotionTasksModal = useModal();

  const onAddPromotionTaskModalOpen = () =>
    promotionTasksModal.show((p) => <CreatePromotionTaskModal {...p} namespace={name || ''} />);

  const onDeletePromotionTaskModalOpen = (promotionTask: PromotionTask) =>
    confirm({
      title: (
        <Flex align='center'>
          <FontAwesomeIcon icon={faTrash} className='mr-2' />
          Delete Promotion Task
        </Flex>
      ),
      content: (
        <p>
          Are you sure you want to delete PromotionTask <b>{promotionTask?.metadata?.name}</b>?
        </p>
      ),
      onOk: () => {
        const manifest = new TextEncoder().encode(
          PromotionTaskManifestsGen.v1alpha1(promotionTask)
        );
        deletePromotionTaskMutation.mutate(
          { manifest },
          { onSuccess: () => listPromotionTasksQuery.refetch() }
        );
      },
      hide: () => {}
    });

  const onEditPromotionTaskModalOpen = (promotionTask: PromotionTask) =>
    promotionTasksModal.show((p) => (
      <EditPromotionTaskModal {...p} promotionTask={promotionTask} />
    ));

  return (
    <div className='p-4'>
      <Table<PromotionTask>
        dataSource={listPromotionTasksQuery.data?.promotionTasks}
        loading={listPromotionTasksQuery.isFetching}
        locale={{
          emptyText: (
            <>
              This project does not have any Promotion Tasks. Read more about PromotionTasks{' '}
              <a
                href='https://docs.kargo.io/user-guide/reference-docs/promotion-tasks/'
                target='_blank'
              >
                here
              </a>
              .
            </>
          )
        }}
        columns={[
          {
            title: 'Creation Date',
            width: 200,
            render: (_, template) => {
              const date = timestampDate(template.metadata?.creationTimestamp);

              return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
            }
          },
          {
            title: 'Name',
            render: (_, r) => r.metadata?.name
          },
          {
            title: (
              <div className='text-right'>
                <Button
                  type='primary'
                  className='ml-auto text-xs font-semibold'
                  icon={<FontAwesomeIcon icon={faPlus} />}
                  onClick={onAddPromotionTaskModalOpen}
                >
                  ADD PROMOTION TASK
                </Button>
              </div>
            ),
            render: (_, template) => (
              <Flex gap={8} justify='end'>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} />}
                  onClick={() => onEditPromotionTaskModalOpen(template)}
                >
                  Edit
                </Button>
                <Button
                  icon={<FontAwesomeIcon icon={faTrash} />}
                  danger
                  onClick={() => onDeletePromotionTaskModalOpen(template)}
                >
                  Delete
                </Button>
              </Flex>
            )
          }
        ]}
      />
    </div>
  );
};
