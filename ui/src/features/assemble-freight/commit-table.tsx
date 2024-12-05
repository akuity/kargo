import { Radio, Table } from 'antd';

import { DiscoveredCommit } from '@ui/gen/v1alpha1/generated_pb';
import { k8sApiMachineryTimestampDate } from '@ui/utils/connectrpc-extension';

import { TruncatedCopyable } from './truncated-copyable';

export const CommitTable = ({
  commits,
  selected,
  select
}: {
  commits: DiscoveredCommit[];
  selected: DiscoveredCommit | undefined;
  select: (commit?: DiscoveredCommit) => void;
}) => {
  return (
    <>
      <Table
        dataSource={commits}
        columns={[
          {
            render: (record: DiscoveredCommit) => (
              <Radio
                checked={selected === record}
                onClick={() => select(selected === record ? undefined : record)}
              />
            )
          },
          {
            title: 'ID',
            render: (record: DiscoveredCommit) => <TruncatedCopyable text={record?.id} />
          },
          {
            title: 'Message',
            render: (record: DiscoveredCommit) => (
              <div style={{ maxWidth: '250px' }}>{record.subject}</div>
            )
          },
          {
            title: 'Author',
            render: (record: DiscoveredCommit) => record.author
          },
          {
            title: 'Committer',
            render: (record: DiscoveredCommit) => record.committer
          },
          {
            title: 'Date',
            render: (record: DiscoveredCommit) =>
              k8sApiMachineryTimestampDate(record.creatorDate)?.toLocaleString()
          }
        ]}
      />
    </>
  );
};
