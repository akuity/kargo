import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Table } from 'antd';
import { format } from 'date-fns';
import { useParams } from 'react-router-dom';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { AnalysisTemplate } from '@ui/gen/rollouts/api/v1alpha1/generated_pb';
import {
  deleteAnalysisTemplate,
  listAnalysisTemplates
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { CreateAnalysisTemplateModal } from './create-analysis-template-modal';
import { EditAnalysisTemplateModal } from './edit-analysis-template-modal';

export const AnalysisTemplatesList = () => {
  const { name } = useParams();
  const confirm = useConfirmModal();

  const { data, isLoading, refetch } = useQuery(listAnalysisTemplates, { project: name });
  const { show: showEdit } = useModal();
  const { show: showCreate } = useModal((p) => (
    <CreateAnalysisTemplateModal {...p} namespace={name || ''} />
  ));
  const { mutate: deleteTemplate, isPending: isDeleting } = useMutation(deleteAnalysisTemplate, {
    onSuccess: () => refetch()
  });

  return (
    <div className='p-4'>
      <Table<AnalysisTemplate>
        dataSource={data?.analysisTemplates}
        pagination={{ hideOnSinglePage: true }}
        rowKey={(i) => i.metadata?.name || ''}
        loading={isLoading}
        expandable={descriptionExpandable()}
      >
        <Table.Column<AnalysisTemplate>
          title='Creation Date'
          width={200}
          render={(_, template) => {
            const date = timestampDate(template.metadata?.creationTimestamp);
            return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
          }}
        />
        <Table.Column<AnalysisTemplate> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<AnalysisTemplate>
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
                    <EditAnalysisTemplateModal
                      {...p}
                      templateName={template.metadata?.name || ''}
                      projectName={name || ''}
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
                        Delete Analysis Template
                      </div>
                    ),
                    content: (
                      <p>
                        Are you sure you want to delete AnalysisTemplate{' '}
                        <b>{template?.metadata?.name}</b>?
                      </p>
                    ),
                    onOk: () => {
                      deleteTemplate({ project: name || '', name: template?.metadata?.name || '' });
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
    </div>
  );
};
