import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Table } from 'antd';
import { format } from 'date-fns';

import { useModal } from '@ui/features/common/modal/use-modal';
import { V1Secret } from '@ui/gen/api/v2/models';
import { useListProjectAPITokens, useListSystemAPITokens } from '@ui/gen/api/v2/rbac/rbac';

import { CreateAPITokenModal } from './create-api-token-modal';
import { DeleteAPITokenButton } from './delete-api-token-button';

type Props = {
  project?: string;
  systemLevel?: boolean;
};

export const APITokensList = ({ project = '', systemLevel = false }: Props) => {
  const systemTokensQuery = useListSystemAPITokens(undefined, { query: { enabled: systemLevel } });
  const projectTokensQuery = useListProjectAPITokens(project, undefined, {
    query: { enabled: !systemLevel && !!project }
  });
  const listAPITokensQuery = systemLevel ? systemTokensQuery : projectTokensQuery;

  const { show } = useModal((p) => (
    <CreateAPITokenModal {...p} project={project} systemLevel={systemLevel} />
  ));

  return (
    <Card
      title='API Tokens'
      type='inner'
      className='min-h-full'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={() => show()}>
          Create API Token
        </Button>
      }
    >
      <Table<V1Secret>
        className='my-2 overflow-x-auto'
        dataSource={listAPITokensQuery.data?.data?.items || []}
        rowKey={(record: V1Secret) => record?.metadata?.name || ''}
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
        loading={listAPITokensQuery.isLoading}
      >
        <Table.Column
          title='Creation Date'
          dataIndex={['metadata', 'creationTimestamp']}
          render={(_, template: V1Secret) => {
            const ts = template.metadata?.creationTimestamp;
            if (!ts) return '';
            const date = new Date(ts);
            return isNaN(date.getTime()) ? '' : format(date, 'MMM do yyyy HH:mm:ss');
          }}
          width={220}
        />
        <Table.Column<V1Secret> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<V1Secret>
          title='Role'
          dataIndex={['metadata', 'annotations', 'kubernetes.io/service-account.name']}
        />
        <Table.Column<V1Secret>
          title='Last usage'
          dataIndex={['metadata', 'labels', 'kubernetes.io/legacy-token-last-used']}
          width={140}
        />
        <Table.Column<V1Secret>
          render={(_, record) => (
            <DeleteAPITokenButton
              name={record.metadata?.name || ''}
              project={project}
              systemLevel={systemLevel}
            />
          )}
          width={100}
        />
      </Table>
    </Card>
  );
};
