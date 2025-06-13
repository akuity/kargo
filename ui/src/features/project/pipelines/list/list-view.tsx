import { Table } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import {
  catalogueFreights,
  catalogueFreightVersions
} from '@ui/features/project/pipelines/freight/source-catalogue-utils';
import { useEventsWatcher } from '@ui/features/project/pipelines/graph/use-events-watcher';
import { Freight, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

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
  warehouses: Warehouse[];
  freights: Freight[];
  className?: string;
  project: string;
};

export const PipelineListView = (props: PipelineListViewProps) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  useEventsWatcher(props.project);

  const catalogouedFreights = useMemo(() => catalogueFreights(props.freights), [props.freights]);

  const cataloguedFreightVersions = useMemo(
    () => catalogueFreightVersions(props.freights),
    [props.freights]
  );

  const [filters, setFilters] = useState<Filter>({
    stage: '',
    phase: [],
    health: [],
    version: {}
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
      <AppliedFilters className='px-2 py-2' />
      <div className={classNames(props.className, 'px-2')}>
        <Table
          dataSource={filteredStages}
          rowKey={(stage) => `${stage?.metadata?.name}-${stage?.status?.observedGeneration}`}
          size='small'
          columns={[
            stageColumn(filters),
            phaseColumn(filters),
            healthColumn(filters),
            versionColumn(
              filters,
              catalogouedFreights,
              cataloguedFreightVersions,
              dictionaryContext?.freightById || {}
            ),
            lastPromotionColumn(filters),
            actionColumn({
              onPromote: (stage) => actionContext?.actPromote(IAction.PROMOTE, stage)
            })
          ]}
        />
      </div>
    </FilterContext.Provider>
  );
};
