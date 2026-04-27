import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Space, Table } from 'antd';
import { format } from 'date-fns';
import { useParams } from 'react-router-dom';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { descriptionExpandable } from '@ui/features/common/description-expandable';
import { useModal } from '@ui/features/common/modal/use-modal';
import { RolloutsAnalysisTemplate } from '@ui/gen/api/v2/models';
import {
  useDeleteAnalysisTemplate,
  useListAnalysisTemplates
} from '@ui/gen/api/v2/verifications/verifications';

import { CreateAnalysisTemplateModal } from './create-analysis-template-modal';
import { EditAnalysisTemplateModal } from './edit-analysis-template-modal';

export const AnalysisTemplatesSettings = () => {
  const { name } = useParams();
  const confirm = useConfirmModal();

  const { data, isLoading, refetch } = useListAnalysisTemplates(name || '');
  const { show: showEdit } = useModal();
  const { show: showCreate } = useModal((p) => (
    <CreateAnalysisTemplateModal {...p} namespace={name || ''} />
  ));
  const { mutate: deleteTemplate, isPending: isDeleting } = useDeleteAnalysisTemplate({
    mutation: { onSuccess: () => refetch() }
  });

  return (
    <Card
      title='Analysis Templates'
      type='inner'
      className='min-h-full'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => showCreate()}>
          Add Template
        </Button>
      }
    >
      <Table<RolloutsAnalysisTemplate>
        dataSource={data?.data?.items}
        pagination={{ hideOnSinglePage: true }}
        rowKey={(i) => i.metadata?.name || ''}
        loading={isLoading}
        expandable={descriptionExpandable()}
        className='my-2'
      >
        <Table.Column<RolloutsAnalysisTemplate>
          title='Creation Date'
          width={200}
          render={(_, template) => {
            const ts = template.metadata?.creationTimestamp;
            if (!ts) return '';
            const date = new Date(ts);
            return isNaN(date.getTime()) ? '' : format(date, 'MMM do yyyy HH:mm:ss');
          }}
        />
        <Table.Column<RolloutsAnalysisTemplate> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<RolloutsAnalysisTemplate>
          width={150}
          render={(_, template) => (
            <Space>
              <Button
                icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
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
                size='small'
                color='default'
                variant='filled'
              >
                Edit
              </Button>
              <Button
                icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                loading={isDeleting}
                color='danger'
                variant='filled'
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
                      deleteTemplate({
                        project: name || '',
                        analysisTemplate: template?.metadata?.name || ''
                      });
                    },
                    hide: () => {}
                  });
                }}
                size='small'
              >
                Delete
              </Button>
            </Space>
          )}
        />
      </Table>
    </Card>
  );
};
