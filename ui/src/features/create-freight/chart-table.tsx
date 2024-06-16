import { Radio, Table } from 'antd';

export const ChartTable = ({
  versions,
  selected,
  select
}: {
  versions: string[];
  selected: string | undefined;
  select: (version?: string) => void;
}) => {
  return (
    <Table
      dataSource={versions.map((version) => ({ version }))}
      columns={[
        {
          width: '50px',
          render: (record: { version: string }) => (
            <Radio
              checked={selected === record.version}
              onClick={() => select(selected === record.version ? undefined : record.version)}
            />
          )
        },
        { title: 'Version', dataIndex: 'version' }
      ]}
    />
  );
};
