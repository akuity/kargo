import { Radio, Table } from 'antd';
import { useState } from 'react';

import { DiscoveredCommit } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { TruncatedCopyable } from './truncated-copyable';

export const CommitTable = ({
  commits,
  selected,
  select,
  show
}: {
  commits: DiscoveredCommit[];
  selected: DiscoveredCommit | undefined;
  select: (commit?: DiscoveredCommit) => void;
  show?: boolean;
}) => {
  const [page, setPage] = useState(1);

  if (!show) {
    return null;
  }

  return (
    <Table
      dataSource={commits}
      pagination={{ current: page, onChange: (page) => setPage(page) }}
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
          render: (record: DiscoveredCommit) => timestampDate(record.creatorDate)?.toLocaleString()
        }
      ]}
    />
  );
};
