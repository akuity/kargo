import { Radio, Table } from 'antd';
import { useLayoutEffect, useRef, useState } from 'react';

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
  const [page, setPage] = useState(1);
  const pageSize = 10;
  const lastSelectedVersionRef = useRef<string | undefined>(undefined);

  useLayoutEffect(() => {
    if (!selected) {
      setPage(1);
      lastSelectedVersionRef.current = undefined;
      return;
    }
    const key = selected;
    if (lastSelectedVersionRef.current === key) return;
    lastSelectedVersionRef.current = key;
    const idx = versions.findIndex((v) => v === key);
    if (idx >= 0) {
      const nextPage = Math.floor(idx / pageSize) + 1;
      if (nextPage !== page) setPage(nextPage);
    }
  }, [selected, versions]);

  if (!show) {
    return null;
  }

  return (
    <Table
      dataSource={versions.map((version) => ({ version }))}
      pagination={{
        current: page,
        onChange: (page) => setPage(page),
        pageSize
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
