import { Table } from 'antd';
import { ColumnType } from 'antd/es/table'; // Add missing import

import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

const renderColumn = (title: string, key: keyof PolicyRule): ColumnType<PolicyRule> => ({
  title,
  key,
  render: (v: PolicyRule) => <>{(v[key] as string[]).join(',')}</>,
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
      pagination={false}
      className='h-full w-full mb-10'
      columns={[
        renderColumn('Verbs', 'verbs'),
        renderColumn('API Groups', 'apiGroups'),
        renderColumn('Resources', 'resources'),
        renderColumn('Resource Names', 'resourceNames'),
        renderColumn('Non-Resource URLs', 'nonResourceURLs'),
        {
          key: 'actions',
          render: (rule) =>
            setRules && (
              <div
                className='text-xs text-gray-400 font-semibold cursor-pointer ml-auto'
                onClick={() => {
                  setRules(rules.filter((r) => r !== rule));
                }}
              >
                REMOVE
              </div>
            )
        }
      ]}
    />
  );
};
