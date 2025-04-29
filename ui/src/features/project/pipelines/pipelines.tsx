import { useQuery } from '@connectrpc/connect-query';
import { useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { mapToNames } from '@ui/features/common/utils';
import FreightDetails from '@ui/features/freight/freight-details';
import WarehouseDetails from '@ui/features/project/pipelines/warehouse/warehouse-details';
import CreateStage from '@ui/features/stage/create-stage';
import CreateWarehouse from '@ui/features/stage/create-warehouse/create-warehouse';
import StageDetails from '@ui/features/stage/stage-details';
import { getColors } from '@ui/features/stage/utils';
import { listStages } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { FreightList } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Project, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { ActionContext } from './context/action-context';
import { DictionaryContext } from './context/dictionary-context';
import {
  FreightTimelineControllerContext,
  FreightTimelineControllerContextType
} from './context/freight-timeline-controller-context';
import { FreightTimeline } from './freight/freight-timeline';
import { Graph } from './graph/graph';
import { GraphFilters } from './graph-filters';
import { Promote } from './promotion/promote';
import { Promotion } from './promotion/promotion';
import { useAction } from './use-action';
import { useFreightById } from './use-freight-by-id';
import { useFreightInStage } from './use-freight-in-stage';
import { useGetFreight } from './use-get-freight';
import { useGetWarehouse } from './use-get-warehouse';
import { usePersistPreferredFilter } from './use-persist-filters';
import { useStageAutoPromotionMap } from './use-stage-auto-promotion-map';
import { useStageByName } from './use-stage-by-name';
import { useSubscribersByStage } from './use-subscribers-by-stage';

import '@xyflow/react/dist/style.css';

export const Pipelines = (props: {
  project: Project;
  stageName?: string;
  promotionId?: string;
  promote?: {
    freight: string;
    stage: string;
  };
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
  warehouseName?: string;
  freightName?: string;
  warehouses: Warehouse[];
  freights: {
    [key: string]: FreightList;
  };
  refetchFreights: () => void;
}) => {
  const navigate = useNavigate();

  const action = useAction();

  const projectName = props.project?.metadata?.name;

  const listStagesQuery = useQuery(listStages, { project: projectName });

  const loading = listStagesQuery.isLoading;

  const stageDetails =
    props.stageName &&
    listStagesQuery?.data?.stages?.find((s) => s?.metadata?.name === props.stageName);

  const warehouseColorMap = useMemo(
    () => getColors(props.project?.metadata?.name || '', props.warehouses, 'warehouses'),
    [props.project, props.warehouses]
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
    showColors: false,
    warehouses: [],
    hideUnusedFreights: false,
    stackedNodesParents: []
  });

  usePersistPreferredFilter(projectName || '', preferredFilter, setPreferredFilter);

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const freightInStages = useFreightInStage(listStagesQuery.data?.stages || []);
  const freightById = useFreightById(props.freights?.['']?.freight || []);
  const stageAutoPromotionMap = useStageAutoPromotionMap(props.project);
  const subscribersByStage = useSubscribersByStage(listStagesQuery.data?.stages || []);
  const stageByName = useStageByName(listStagesQuery.data?.stages || []);
  const warehouseDrawer = useGetWarehouse(props.warehouses, props.warehouseName);
  const freightDrawer = useGetFreight(props.freights?.[''], props.freightName);

  if (loading) {
    return <LoadingState />;
  }

  return (
    <ActionContext.Provider value={action}>
      <FreightTimelineControllerContext.Provider
        value={{
          viewingFreight,
          setPreferredFilter,
          preferredFilter,
          setViewingFreight
        }}
      >
        <DictionaryContext.Provider
          value={{
            freightInStages,
            freightById,
            stageAutoPromotionMap,
            subscribersByStage,
            stageByName
          }}
        >
          <ColorContext.Provider value={{ stageColorMap, warehouseColorMap }}>
            <FreightTimeline
              freights={props.freights?.['']?.freight || []}
              project={projectName || ''}
            />

            <div className='w-full h-full relative'>
              <GraphFilters
                warehouses={props.warehouses}
                stages={listStagesQuery.data?.stages || []}
              />
              <Graph
                project={props.project.metadata?.name || ''}
                warehouses={props.warehouses}
                stages={listStagesQuery.data?.stages || []}
              />
            </div>

            {!!freightDrawer && (
              <FreightDetails freight={freightDrawer} refetchFreight={props.refetchFreights} />
            )}

            {!!warehouseDrawer && (
              <WarehouseDetails
                warehouse={warehouseDrawer}
                refetchFreight={props.refetchFreights}
              />
            )}

            {!!stageDetails && <StageDetails stage={stageDetails} />}

            {!!props.promotionId && (
              <Promotion
                visible
                hide={() => navigate(generatePath(paths.project, { name: projectName }))}
                promotionId={props.promotionId}
                project={projectName || ''}
              />
            )}

            {!!props.promote && (
              <Promote
                visible
                hide={() => navigate(generatePath(paths.project, { name: projectName }))}
                freight={freightById?.[props.promote.freight]}
                stage={stageByName?.[props.promote.stage]}
              />
            )}

            {props.creatingStage && (
              <CreateStage
                project={props.project?.metadata?.name}
                warehouses={mapToNames(props.warehouses || [])}
                stages={mapToNames(listStagesQuery.data?.stages || [])}
              />
            )}

            <CreateWarehouse
              visible={Boolean(props.creatingWarehouse)}
              hide={() =>
                navigate(generatePath(paths.project, { name: props.project?.metadata?.name }))
              }
            />
          </ColorContext.Provider>
        </DictionaryContext.Provider>
      </FreightTimelineControllerContext.Provider>
    </ActionContext.Provider>
  );
};
