import { ColumnType } from 'antd/es/table';

import { Stage } from '@ui/gen/api/v2/models';

import { VersionCell } from './version-cell';

export const versionColumn = (): ColumnType<Stage> => ({
  title: 'Version',
  render: (_, stage) => <VersionCell stage={stage} />
});
