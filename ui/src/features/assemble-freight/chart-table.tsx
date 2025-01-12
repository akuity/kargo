import { Radio, Table } from 'antd';
import { useState } from 'react';

import { calculatePageForSelectedRow } from '@ui/utils/pagination';

export const ChartTable = ({
  versions,
  selected,
  select
}: {
  versions: string[];
  selected: string | undefined;
  select: (version?: string) => void;
}) => {
  const [defaultPage] = useState<number>(() =>
    calculatePageForSelectedRow(selected, versions, (version) => version)
  );

  return (
    <Table
      dataSource={versions.map((version) => ({ version }))}
      pagination={{ defaultCurrent: defaultPage }}
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
