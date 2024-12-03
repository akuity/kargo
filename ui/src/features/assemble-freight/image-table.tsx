import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Radio, Table } from 'antd';

import { DiscoveredImageReference } from '@ui/gen/v1alpha1/generated_pb';
import { k8sApiMachineryTimestampDate } from '@ui/utils/connectrpc-extension';

import { TruncatedCopyable } from './truncated-copyable';

export const ImageTable = ({
  references,
  selected,
  select
}: {
  references: DiscoveredImageReference[];
  selected: DiscoveredImageReference | undefined;
  select: (reference?: DiscoveredImageReference) => void;
}) => (
  <>
    <Table
      dataSource={references}
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
            k8sApiMachineryTimestampDate(record.createdAt).toLocaleString()
        }
      ]}
    />
  </>
);
