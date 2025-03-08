import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Table } from 'antd';
import { format } from 'date-fns';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteClusterAnalysisTemplate,
  listClusterAnalysisTemplates
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { ClusterAnalysisTemplate } from '@ui/gen/stubs/rollouts/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { CreateClusterAnalysisTemplateModal } from './create-cluster-analysis-template-modal';
import { EditClusterAnalysisTemplateModal } from './edit-cluster-analysis-template-modal';

export const ClusterAnalysisTemplatesList = () => {
  const confirm = useConfirmModal();

  const { data, isLoading, refetch } = useQuery(listClusterAnalysisTemplates);

  const { show: showEdit } = useModal();

  const { show: showCreate } = useModal((p) => <CreateClusterAnalysisTemplateModal {...p} />);

  const { mutate: deleteTemplate, isPending: isDeleting } = useMutation(
    deleteClusterAnalysisTemplate,
    {
      onSuccess: () => refetch()
    }
  );

  return (
    <Table<ClusterAnalysisTemplate>
      dataSource={data?.clusterAnalysisTemplates}
      pagination={{ hideOnSinglePage: true }}
      rowKey={(i) => i.metadata?.name || ''}
      loading={isLoading}
      expandable={descriptionExpandable()}
      className='w-full'
    >
      <Table.Column<ClusterAnalysisTemplate>
        title='Creation Date'
        width={200}
        render={(_, template) => {
          const date = timestampDate(template.metadata?.creationTimestamp);
          return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
        }}
      />
      <Table.Column<ClusterAnalysisTemplate> title='Name' dataIndex={['metadata', 'name']} />
      <Table.Column<ClusterAnalysisTemplate>
        width={260}
        title={
          <div className='text-right'>
            <Button
              type='primary'
              className='ml-auto text-xs font-semibold'
              icon={<FontAwesomeIcon icon={faPlus} />}
              onClick={() => showCreate()}
            >
              ADD TEMPLATE
            </Button>
          </div>
        }
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
            >
              Edit
            </Button>
            <Button
              icon={<FontAwesomeIcon icon={faTrash} />}
              danger
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
                    deleteTemplate({ name: template?.metadata?.name || '' });
                  },
                  hide: () => {}
                });
              }}
            >
              Delete
            </Button>
          </div>
        )}
      />
    </Table>
  );
};
