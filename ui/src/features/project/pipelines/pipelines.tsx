import { useQuery } from '@connectrpc/connect-query';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex } from 'antd';
import { useMemo, useState } from 'react';
import { generatePath, Link, useNavigate, useParams } from 'react-router-dom';

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
import {
  listStages,
  listWarehouses,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { FreightList } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Project } from '@ui/gen/api/v1alpha1/generated_pb';

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
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { stageName, promotionId, freight, stage, warehouseName, freightName } = useParams();

  const projectName = props.project?.metadata?.name;

  const getFreightQuery = useQuery(queryFreight, { project: projectName });

  const listWarehousesQuery = useQuery(listWarehouses, {
    project: projectName
  });

  const listStagesQuery = useQuery(listStages, { project: projectName });

  const loading =
    getFreightQuery.isLoading || listWarehousesQuery.isLoading || listStagesQuery.isLoading;

  const promote = freight && stage ? { freight, stage } : undefined;

  const navigate = useNavigate();

  const action = useAction();

  const stageDetails =
    stageName && listStagesQuery.data?.stages.find((s) => s?.metadata?.name === stageName);

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
    sources: [],
    timerange: 'all-time',
    showColors: true,
    warehouses: [],
    hideUnusedFreights: false,
    stackedNodesParents: [],
    hideSubscriptions: {}
  });

  usePersistPreferredFilter(projectName || '', preferredFilter, setPreferredFilter);

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const freightInStages = useFreightInStage(listStagesQuery.data?.stages || []);
  const freightById = useFreightById(getFreightQuery.data?.groups?.['']?.freight || []);
  const stageAutoPromotionMap = useStageAutoPromotionMap(props.project);
  const subscribersByStage = useSubscribersByStage(listStagesQuery.data?.stages || []);
  const stageByName = useStageByName(listStagesQuery.data?.stages || []);
  const warehouseDrawer = useGetWarehouse(
    listWarehousesQuery.data?.warehouses || [],
    warehouseName
  );
  const freightDrawer = useGetFreight(
    getFreightQuery.data?.groups?.[''] as FreightList,
    freightName
  );

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
              freights={getFreightQuery.data?.groups?.['']?.freight || []}
              project={projectName || ''}
            />

            <div className='w-full h-full relative'>
              <Flex justify='space-between' gap={24} className='absolute z-10 top-2 right-2 left-2'>
                <GraphFilters
                  warehouses={listWarehousesQuery.data?.warehouses || []}
                  stages={listStagesQuery.data?.stages || []}
                />
                <Dropdown
                  trigger={['click']}
                  menu={{
                    items: [
                      {
                        key: '0',
                        label: (
                          <Link
                            to={generatePath(paths.createStage, {
                              name: props.project.metadata?.name
                            })}
                          >
                            Stage
                          </Link>
                        )
                      },
                      {
                        key: '1',
                        label: (
                          <Link
                            to={generatePath(paths.createWarehouse, {
                              name: props.project.metadata?.name
                            })}
                          >
                            Warehouse
                          </Link>
                        )
                      },
                      {
                        key: '2',
                        label: 'Freight',
                        children: listWarehousesQuery.data?.warehouses?.map((warehouse) => ({
                          key: warehouse?.metadata?.name || '',
                          label: warehouse?.metadata?.name || '',
                          onClick: () => {
                            navigate(
                              generatePath(paths.warehouse, {
                                name: props.project.metadata?.name,
                                warehouseName: warehouse?.metadata?.name || '',
                                tab: 'create-freight'
                              })
                            );
                          }
                        }))
                      }
                    ]
                  }}
                >
                  <Button icon={<FontAwesomeIcon icon={faPlus} />}>Create</Button>
                </Dropdown>
              </Flex>
              <Graph
                project={props.project.metadata?.name || ''}
                warehouses={listWarehousesQuery.data?.warehouses || []}
                stages={listStagesQuery.data?.stages || []}
              />
            </div>

            {!!freightDrawer && (
              <FreightDetails freight={freightDrawer} refetchFreight={getFreightQuery.refetch} />
            )}

            {!!warehouseDrawer && (
              <WarehouseDetails
                warehouse={warehouseDrawer}
                refetchFreight={getFreightQuery.refetch}
              />
            )}

            {!!stageDetails && <StageDetails stage={stageDetails} />}

            {!!promotionId && (
              <Promotion
                visible
                hide={() => navigate(generatePath(paths.project, { name: projectName }))}
                promotionId={promotionId}
                project={projectName || ''}
              />
            )}

            {!!promote && (
              <Promote
                visible
                hide={() => navigate(generatePath(paths.project, { name: projectName }))}
                freight={freightById?.[promote.freight]}
                stage={stageByName?.[promote.stage]}
              />
            )}

            {props.creatingStage && (
              <CreateStage
                project={props.project?.metadata?.name}
                warehouses={mapToNames(listWarehousesQuery.data?.warehouses || [])}
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
