import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Table } from 'antd';
import { format } from 'date-fns';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { RolloutsClusterAnalysisTemplate } from '@ui/gen/api/v2/models';
import {
  useDeleteClusterAnalysisTemplate,
  useListClusterAnalysisTemplates
} from '@ui/gen/api/v2/verifications/verifications';

import { CreateClusterAnalysisTemplateModal } from './create-cluster-analysis-template-modal';
import { EditClusterAnalysisTemplateModal } from './edit-cluster-analysis-template-modal';

export const ClusterAnalysisTemplatesList = () => {
  const confirm = useConfirmModal();

  const { data, isLoading, refetch } = useListClusterAnalysisTemplates();

  const { show: showEdit } = useModal();

  const { show: showCreate } = useModal((p) => <CreateClusterAnalysisTemplateModal {...p} />);

  const { mutate: deleteTemplate, isPending: isDeleting } = useDeleteClusterAnalysisTemplate({
    mutation: {
      onSuccess: () => refetch()
    }
  });

  return (
    <Card
      title='Cluster Analysis Templates'
      type='inner'
      className='min-h-full'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => showCreate()}>
          Add Template
        </Button>
      }
    >
      <Table<RolloutsClusterAnalysisTemplate>
        dataSource={data?.data?.items}
        pagination={{ hideOnSinglePage: true }}
        rowKey={(i) => i.metadata?.name || ''}
        loading={isLoading}
        expandable={descriptionExpandable()}
        className='w-full'
      >
        <Table.Column<RolloutsClusterAnalysisTemplate>
          title='Creation Date'
          width={200}
          render={(_, template) => {
            const ts = template.metadata?.creationTimestamp;
            if (!ts) return '';
            const date = new Date(ts);
            return isNaN(date.getTime()) ? '' : format(date, 'MMM do yyyy HH:mm:ss');
          }}
        />
        <Table.Column<RolloutsClusterAnalysisTemplate>
          title='Name'
          dataIndex={['metadata', 'name']}
        />
        <Table.Column<RolloutsClusterAnalysisTemplate>
          width={260}
          render={(_, template) => (
            <div className='flex gap-2 justify-end'>
              <Button
                icon={<FontAwesomeIcon icon={faPencil} />}
                className='mr-2 ml-auto'
                onClick={() => {
                  showEdit((p) => (
                    <EditClusterAnalysisTemplateModal
                      {...p}
                      templateName={template.metadata?.name || ''}
                    />
                  ));
                }}
                size='small'
                color='default'
                variant='filled'
              >
                Edit
              </Button>
              <Button
                icon={<FontAwesomeIcon icon={faTrash} />}
                loading={isDeleting}
                onClick={() => {
                  confirm({
                    title: (
                      <div className='flex items-center'>
                        <FontAwesomeIcon icon={faTrash} className='mr-2' />
                        Delete Cluster Analysis Template
                      </div>
                    ),
                    content: (
                      <p>
                        Are you sure you want to delete ClusterAnalysisTemplate{' '}
                        <b>{template?.metadata?.name}</b>?
                      </p>
                    ),
                    onOk: () => {
                      deleteTemplate({
                        clusterAnalysisTemplate: template?.metadata?.name || ''
                      });
                    },
                    hide: () => {}
                  });
                }}
                size='small'
                color='danger'
                variant='filled'
              >
                Delete
              </Button>
            </div>
          )}
        />
      </Table>
    </Card>
  );
};
