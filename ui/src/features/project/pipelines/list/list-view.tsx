import { Card, Table } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useEventsWatcher } from '@ui/features/project/pipelines/graph/use-events-watcher';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

import { actionColumn } from './columns/action-column';
import { healthColumn } from './columns/health-column';
import { lastPromotionColumn } from './columns/last-promotion-column';
import { phaseColumn } from './columns/phase-column';
import { stageColumn } from './columns/stage-column';
import { versionColumn } from './columns/version-column';

type PipelineListViewProps = {
  stages: Stage[];
  warehouses: WarehouseExpanded[];
  freights: Freight[];
  className?: string;
  project: string;
};

export const PipelineListView = (props: PipelineListViewProps) => {
  const actionContext = useActionContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  useEventsWatcher(
    props.project,
    undefined,
    freightTimelineControllerContext?.preferredFilter?.warehouses || []
  );

  const filteredStages = useMemo(() => {
    const filterWarehouses = freightTimelineControllerContext?.preferredFilter?.warehouses || [];
    const search = freightTimelineControllerContext?.stageSearch?.trim().toLowerCase() || '';

    let stages = props.stages;

    if (filterWarehouses.length) {
      stages = stages.filter((stage) =>
        stage.spec?.requestedFreight?.find((f) => filterWarehouses.includes(f?.origin?.name || ''))
      );
    }

    if (search) {
      stages = stages.filter((stage) => stage.metadata?.name?.toLowerCase().includes(search));
    }

    return stages;
  }, [
    freightTimelineControllerContext?.preferredFilter?.warehouses,
    freightTimelineControllerContext?.stageSearch,
    props.stages
  ]);

  return (
    <Card className={classNames(props.className, 'm-2')} size='small'>
      <Table
        dataSource={filteredStages}
        rowKey={(stage) => `${stage?.metadata?.name}-${stage?.status?.observedGeneration}`}
        size='small'
        columns={[
          stageColumn(),
          phaseColumn(),
          healthColumn(),
          versionColumn(),
          lastPromotionColumn(),
          actionColumn({
            onPromote: (stage) => actionContext?.actPromote(IAction.PROMOTE, stage)
          })
        ]}
      />
    </Card>
  );
};
