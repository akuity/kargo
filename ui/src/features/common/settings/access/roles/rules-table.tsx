import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Table } from 'antd';
import { ColumnType } from 'antd/es/table'; // Add missing import

import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

const renderColumn = (title: string, key: keyof PolicyRule): ColumnType<PolicyRule> => ({
  title,
  key,
  render: (v: PolicyRule) => <div key={key}>{((v[key] || []) as string[]).join(', ')}</div>,
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
  rules: PolicyRule[];
  setRules?: (rules: PolicyRule[]) => void;
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
