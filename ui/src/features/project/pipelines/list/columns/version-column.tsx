import { ColumnType } from 'antd/es/table';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { VersionCell } from './version-cell';

export const versionColumn = (): ColumnType<Stage> => ({
  title: 'Version',
  render: (_, stage) => <VersionCell stage={stage} />
});
