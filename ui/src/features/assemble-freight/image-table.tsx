import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Radio, Table } from 'antd';
import { useLayoutEffect, useRef, useState } from 'react';

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
  const pageSize = 10;
  const lastSelectedTagRef = useRef<string | undefined>(undefined);

  useLayoutEffect(() => {
    if (!selected) {
      setPage(1);
      lastSelectedTagRef.current = undefined;
      return;
    }
    const key = selected?.tag;
    if (lastSelectedTagRef.current === key) {
      return;
    }
    lastSelectedTagRef.current = key;
    const idx = references.findIndex((r) => r?.tag === key);
    if (idx >= 0) {
      const nextPage = Math.floor(idx / pageSize) + 1;
      if (nextPage !== page) setPage(nextPage);
    }
  }, [selected, references]);

  if (!show) {
    return null;
  }

  return (
    <Table
      dataSource={references}
      pagination={{ current: page, onChange: (page) => setPage(page), pageSize }}
      columns={[
        {
          render: (record: DiscoveredImageReference) => (
            <Radio
              checked={selected?.tag === record?.tag}
              onClick={() => select(selected?.tag === record?.tag ? undefined : record)}
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
          render: (record: DiscoveredImageReference) => {
            // Use OCI annotation for source repository URL
            const sourceUrl = record?.annotations?.['org.opencontainers.image.source'];
            return sourceUrl ? (
              <a href={sourceUrl} target='_blank' rel='noreferrer'>
                {sourceUrl}
              </a>
            ) : (
              <FontAwesomeIcon icon={faQuestionCircle} className='text-gray-400' />
            );
          }
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
