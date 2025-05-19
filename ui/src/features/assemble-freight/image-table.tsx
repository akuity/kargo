import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Radio, Table } from 'antd';
import { useState } from 'react';

import { DiscoveredImageReference } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { TruncatedCopyable } from './truncated-copyable';

export const ImageTable = ({
  references,
  selected,
  select,
  show
}: {
  references: DiscoveredImageReference[];
  selected: DiscoveredImageReference | undefined;
  select: (reference?: DiscoveredImageReference) => void;
  show?: boolean;
}) => {
  const [page, setPage] = useState(1);

  if (!show) {
    return null;
  }

  return (
    <Table
      dataSource={references}
      pagination={{ current: page, onChange: (page) => setPage(page) }}
      columns={[
        {
          render: (record: DiscoveredImageReference) => (
            <Radio
              checked={selected?.tag === record?.tag}
              onClick={() => select(selected === record ? undefined : record)}
            />
          )
        },
        { title: 'Tag', dataIndex: 'tag' },
        {
          title: 'Digest',
          render: (record: DiscoveredImageReference) => <TruncatedCopyable text={record?.digest} />
        },
        {
          title: 'Source Repo',
          render: (record: DiscoveredImageReference) =>
            record?.gitRepoURL ? (
              // Deprecated: Use OCI annotations instead. Will be removed in version 1.7.
              <a href={record?.gitRepoURL} target='_blank' rel='noreferrer'>
                {record?.gitRepoURL}
              </a>
            ) : (
              <FontAwesomeIcon icon={faQuestionCircle} className='text-gray-400' />
            )
        },
        {
          title: 'Created At',
          render: (record: DiscoveredImageReference) =>
            timestampDate(record.createdAt)?.toLocaleString()
        }
      ]}
    />
  );
};
