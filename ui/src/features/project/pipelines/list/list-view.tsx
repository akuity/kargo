import { Card, Table } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useEventsWatcher } from '@ui/features/project/pipelines/graph/use-events-watcher';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

import { AppliedFilters } from './applied-filters';
import { actionColumn } from './columns/action-column';
import { healthColumn } from './columns/health-column';
import { lastPromotionColumn } from './columns/last-promotion-column';
import { phaseColumn } from './columns/phase-column';
import { stageColumn } from './columns/stage-column';
import { versionColumn } from './columns/version-column';
import { Filter, FilterContext } from './context/filter-context';

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

  useEventsWatcher(props.project);

  const [filters, setFilters] = useState<Filter>({
    stage: ''
  });

  const filteredStages = useMemo(() => {
    const filterWarehouses = freightTimelineControllerContext?.preferredFilter?.warehouses || [];

    if (!freightTimelineControllerContext?.preferredFilter?.warehouses?.length) {
      return props.stages;
    }

    return props.stages.filter((stage) => {
      return stage.spec?.requestedFreight?.find((f) =>
        filterWarehouses.includes(f?.origin?.name || '')
      );
    });
  }, [freightTimelineControllerContext?.preferredFilter?.warehouses, props.stages]);

  return (
    <FilterContext.Provider
      value={{
        filters,
        onFilter: setFilters
      }}
    >
      <Card className={classNames(props.className, 'm-2')} size='small'>
        <AppliedFilters className='px-2 pb-4' />
        <Table
          pagination={{ hideOnSinglePage: true }}
          dataSource={filteredStages}
          rowKey={(stage) => `${stage?.metadata?.name}-${stage?.status?.observedGeneration}`}
          size='small'
          columns={[
            stageColumn(filters),
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
    </FilterContext.Provider>
  );
};
