import { Radio, Table } from 'antd';
import { useState } from 'react';

import { DiscoveredCommit } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { calculatePageForSelectedRow } from '@ui/utils/pagination';

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
  const [defaultPage] = useState<number>(() =>
    calculatePageForSelectedRow(selected, commits, (commit) => commit.id)
  );

  return (
    <>
      <Table
        dataSource={commits}
        pagination={{ defaultCurrent: defaultPage }}
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
              timestampDate(record.creatorDate)?.toLocaleString()
          }
        ]}
      />
    </>
  );
};
