import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Table } from 'antd';
import { ColumnType } from 'antd/es/table'; // Add missing import

import { V1PolicyRule } from '@ui/gen/api/v2/models';

const renderColumn = (title: string, key: keyof V1PolicyRule): ColumnType<V1PolicyRule> => ({
  title,
  key,
  render: (v: V1PolicyRule) => <div key={key}>{((v[key] || []) as string[]).join(', ')}</div>,
  onCell: () => ({
    style: {
      padding: '5px'
    }
  }),
  onHeaderCell: () => ({
    style: {
      fontWeight: '500',
      fontSize: '12px',
      padding: '5px',
      textTransform: 'uppercase',
      opacity: 0.5
    }
  })
});

export const RulesTable = ({
  rules,
  setRules
}: {
  rules: V1PolicyRule[];
  setRules?: (rules: V1PolicyRule[]) => void;
}) => {
  return (
    <Table
      dataSource={rules}
      rowKey={(rule) => JSON.stringify(rule)}
      pagination={false}
      className='h-full w-full mb-10 max-w-full'
      columns={[
        renderColumn('Verbs', 'verbs'),
        renderColumn('Resources', 'resources'),
        renderColumn('Resource Names', 'resourceNames'),
        {
          key: 'actions',
          render: (rule) =>
            setRules && (
              <div
                key='actions'
                className='text-xs text-gray-400 font-semibold cursor-pointer ml-auto'
                onClick={() => {
                  setRules(rules.filter((r) => r !== rule));
                }}
              >
                <FontAwesomeIcon icon={faTrash} />
              </div>
            )
        }
      ]}
    />
  );
};
