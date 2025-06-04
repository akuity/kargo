import { Table } from 'antd';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

type PipelineListViewProps = {
  stages: Stage[];
  warehouses: Warehouse[];
};

export const PipelineListView = (props: PipelineListViewProps) => {
  return <Table />;
};
