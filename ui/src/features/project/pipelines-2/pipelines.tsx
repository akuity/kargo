import { useQuery } from '@connectrpc/connect-query';
import { useMemo, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { getColors } from '@ui/features/stage/utils';
import {
  listStages,
  listWarehouses,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Project } from '@ui/gen/api/v1alpha1/generated_pb';

import { DictionaryContext } from './context/dictionary-context';
import {
  FreightTimelineControllerContext,
  FreightTimelineControllerContextType
} from './context/freight-timeline-controller-context';
import { FreightTimeline } from './freight/freight-timeline';
import { Graph } from './graph/graph';
import { useFreightById } from './use-freight-by-id';
import { useFreightInStage } from './use-freight-in-stage';
import { useStageAutoPromotionMap } from './use-stage-auto-promotion-map';

import '@xyflow/react/dist/style.css';

export const Pipelines = (props: { project: Project }) => {
  const projectName = props.project?.metadata?.name;

  const getFreightQuery = useQuery(queryFreight, { project: projectName });

  const listWarehousesQuery = useQuery(listWarehouses, { project: projectName });

  const listStagesQuery = useQuery(listStages, { project: projectName });

  const loading =
    getFreightQuery.isLoading || listWarehousesQuery.isLoading || listStagesQuery.isLoading;

  const warehouseColorMap = useMemo(
    () =>
      getColors(
        props.project?.metadata?.name || '',
        listWarehousesQuery.data?.warehouses || [],
        'warehouses'
      ),
    [props.project, listWarehousesQuery.data?.warehouses]
  );

  const stageColorMap = useMemo(
    () => getColors(props.project?.metadata?.name || '', listStagesQuery.data?.stages || []),
    [props.project, listStagesQuery.data?.stages]
  );

  const [preferredFilter, setPreferredFilter] = useState<
    FreightTimelineControllerContextType['preferredFilter']
  >({
    showAlias: false,
    artifactCarousel: {
      enabled: false
    },
    sources: [],
    timerange: 'all-time',
    showColors: false
  });

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const freightInStages = useFreightInStage(listStagesQuery.data?.stages || []);
  const freightById = useFreightById(getFreightQuery?.data?.groups?.['']?.freight || []);
  const stageAutoPromotionMap = useStageAutoPromotionMap(props.project);

  if (loading) {
    return <LoadingState />;
  }

  return (
    <FreightTimelineControllerContext.Provider
      value={{
        viewingFreight,
        setPreferredFilter,
        preferredFilter,
        setViewingFreight
      }}
    >
      <DictionaryContext.Provider value={{ freightInStages, freightById, stageAutoPromotionMap }}>
        <ColorContext.Provider value={{ stageColorMap, warehouseColorMap }}>
          <FreightTimeline freights={getFreightQuery?.data?.groups?.['']?.freight || []} />

          <div className='w-full h-full'>
            <Graph
              project={props.project.metadata?.name || ''}
              warehouses={listWarehousesQuery.data?.warehouses || []}
              stages={listStagesQuery.data?.stages || []}
            />
          </div>
        </ColorContext.Provider>
      </DictionaryContext.Provider>
    </FreightTimelineControllerContext.Provider>
  );
};
