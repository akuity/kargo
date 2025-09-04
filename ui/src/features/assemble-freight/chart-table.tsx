import { Radio, Table } from 'antd';

import { useDetectPage } from './use-detect-page';

export const ChartTable = ({
  versions,
  selected,
  select,
  show
}: {
  versions: string[];
  selected: string | undefined;
  select: (version?: string) => void;
  show?: boolean;
}) => {
  const [page, setPage] = useDetectPage(versions, selected, !show);

  if (!show) {
    return null;
  }

  return (
    <Table
      dataSource={versions.map((version) => ({ version }))}
      pagination={{
        current: page,
        onChange: (page) => setPage(page)
      }}
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
