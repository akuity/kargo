import { faCheck, faClipboard, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Checkbox, Table } from 'antd';
import { useState } from 'react';

import { DiscoveredImageReference } from '@ui/gen/v1alpha1/generated_pb';

export const ImageTable = ({
  references,
  selected,
  select
}: {
  references: DiscoveredImageReference[];
  selected: DiscoveredImageReference | undefined;
  select: (reference?: DiscoveredImageReference) => void;
}) => {
  // Antd "copyable" Typography does not work with truncated text
  const [copied, setCopied] = useState<string | undefined>(undefined);
  return (
    <>
      <Table
        dataSource={references}
        columns={[
          {
            render: (record: DiscoveredImageReference) => (
              <Checkbox
                checked={selected === record}
                onClick={() => select(selected === record ? undefined : record)}
              />
            )
          },
          { title: 'Tag', dataIndex: 'tag' },
          {
            title: 'Digest',
            render: (record: DiscoveredImageReference) => (
              <div
                className='flex items-center cursor-pointer hover:text-blue-500'
                onClick={() => {
                  if (record.digest) {
                    navigator.clipboard.writeText(record.digest);
                    setCopied(record?.tag);
                    setTimeout(() => setCopied(undefined), 1000);
                  }
                }}
              >
                <FontAwesomeIcon
                  icon={copied === record.tag ? faCheck : faClipboard}
                  className='mr-2 w-3 text-neutral-400'
                />
                <div className='truncate font-mono text-xs' style={{ maxWidth: '200px' }}>
                  {record.digest}
                </div>
              </div>
            )
          },
          {
            title: 'Source Repo',
            render: (record: DiscoveredImageReference) =>
              record?.gitRepoURL ? (
                <a href={record?.gitRepoURL} target='_blank' rel='noreferrer'>
                  {record?.gitRepoURL}
                </a>
              ) : (
                <FontAwesomeIcon icon={faQuestionCircle} className='text-neutral-400' />
              )
          },
          {
            title: 'Created At',
            render: (record: DiscoveredImageReference) =>
              record.createdAt?.toDate().toLocaleString()
          }
        ]}
      />
    </>
  );
};
