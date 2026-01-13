import { timestampDate } from '@bufbuild/protobuf/wkt';
import { useQuery } from '@connectrpc/connect-query';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Table } from 'antd';
import { format } from 'date-fns';

import { useModal } from '@ui/features/common/modal/use-modal';
import { listAPITokens } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { CreateAPITokenModal } from './create-api-token-modal';
import { DeleteAPITokenButton } from './delete-api-token-button';

type Props = {
  project?: string;
  systemLevel?: boolean;
};

export const APITokensList = ({ project = '', systemLevel = false }: Props) => {
  const listAPITokensQuery = useQuery(listAPITokens, { project, systemLevel });

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
      <Table<Secret>
        className='my-2 overflow-x-auto'
        dataSource={listAPITokensQuery.data?.tokenSecrets || []}
        rowKey={(record: Secret) => record?.metadata?.name || ''}
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
        loading={listAPITokensQuery.isLoading}
      >
        <Table.Column
          title='Creation Date'
          dataIndex={['metadata', 'creationTimestamp']}
          render={(_, template) => {
            const date = timestampDate(template.metadata?.creationTimestamp);
            return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
          }}
          width={220}
        />
        <Table.Column<Secret> title='Name' dataIndex={['metadata', 'name']} />
        <Table.Column<Secret>
          title='Role'
          dataIndex={['metadata', 'annotations', 'kubernetes.io/service-account.name']}
        />
        <Table.Column<Secret>
          title='Last usage'
          dataIndex={['metadata', 'labels', 'kubernetes.io/legacy-token-last-used']}
          width={140}
        />
        <Table.Column<Secret>
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
