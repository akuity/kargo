import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex, Result } from 'antd';
import classNames from 'classnames';
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
  useGetProject,
  useListImages,
  useListStages,
  useListWarehouses,
  useQueryFreightsRest
} from '@ui/gen/api/v2/core/core';
import { Freight, Project } from '@ui/gen/api/v2/models';
import { useGetConfig, useGetControllerHeartbeats } from '@ui/gen/api/v2/system/system';

import { ActionContext } from './context/action-context';
import { DictionaryContext } from './context/dictionary-context';
import { FreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { FreightTimeline } from './freight/freight-timeline';
import { Graph } from './graph/graph';
import { GraphFilters } from './graph-filters';
import { Images } from './image-history/images';
import { PipelineListView } from './list/list-view';
import { DndPromotionContext } from './promotion/drag-and-drop/dnd-promotion-context';
import { Promote } from './promotion/promote';
import { Promotion } from './promotion/promotion';
import { useFreightTimelineControllerStore } from './url-params/use-freight-timeline-controller-store';
import { useAction } from './use-action';
import { useFreightById } from './use-freight-by-id';
import { useFreightInStage } from './use-freight-in-stage';
import { useGetFreight } from './use-get-freight';
import { useGetWarehouse } from './use-get-warehouse';
import { usePersistPreferredFilter } from './use-persist-filters';
import { useStageAutoPromotionMap } from './use-stage-auto-promotion-map';
import { useStageByName } from './use-stage-by-name';
import { useSubscribersByStage } from './use-subscribers-by-stage';
import { useSyncFreight } from './use-sync-freight';
import { useWatchFreight } from './use-watch-freight';

import '@xyflow/react/dist/style.css';

export const Pipelines = (props: { creatingStage?: boolean; creatingWarehouse?: boolean }) => {
  const { name, stageName, promotionId, freight, stage, warehouseName, freightName } = useParams();

  if (!name) {
    throw new Error(`undefined project name`);
  }

  const getConfigQuery = useGetConfig();

  const argocdShards = getConfigQuery?.data?.data?.argocdShards;

  const heartbeatsQuery = useGetControllerHeartbeats();
  const heartbeatsByController = heartbeatsQuery.data?.data?.heartbeats;
  const defaultControllerName = heartbeatsQuery.data?.data?.defaultController;

  const projectQuery = useGetProject(name);

  const project = projectQuery.data?.data as Project;

  const projectName = name;

  const listImagesQuery = useListImages(name);

  const listWarehousesQuery = useListWarehouses(projectName);

  const warehouses = listWarehousesQuery?.data?.data?.items || [];

  const [preferredFilter, setPreferredFilter] = useFreightTimelineControllerStore(projectName);

  const getFreightQuery = useQueryFreightsRest(projectName, {
    origins: preferredFilter.warehouses
  });

  const listStagesQuery = useListStages(projectName, {
    freightOrigins: preferredFilter.warehouses
  });

  const stages = listStagesQuery.data?.data?.items || [];

  const loading =
    listStagesQuery.isLoading ||
    projectQuery.isLoading ||
    getFreightQuery.isLoading ||
    listWarehousesQuery.isLoading ||
    getConfigQuery.isLoading;

  const promote = freight && stage ? { freight, stage } : undefined;

  const navigate = useNavigate();

  const action = useAction();

  const stageDetails = stageName && stages.find((s) => s?.metadata?.name === stageName);

  const warehouseColorMap = useMemo(() => {
    if (warehouses.length < 2) {
      return {};
    }
    return getColors(project?.metadata?.name || '', warehouses, 'warehouses');
  }, [project, warehouses]);

  const stageColorMap = useMemo(
    () => getColors(project?.metadata?.name || '', stages),
    [project, stages]
  );

  const pipelineView = preferredFilter.view;

  const setPipelineView = (nextView: 'graph' | 'list') => {
    setPreferredFilter({ ...preferredFilter, view: nextView });
  };

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);
  const [stageSearch, setStageSearch] = useState<string>('');

  const freightInStages = useFreightInStage(stages);
  const freightById = useFreightById(getFreightQuery.data?.data?.groups?.['']?.items || []);
  const stageAutoPromotionMap = useStageAutoPromotionMap(stages);
  const subscribersByStage = useSubscribersByStage(stages);
  const stageByName = useStageByName(stages);
  const warehouseDrawer = useGetWarehouse(warehouses, warehouseName);
  const freightDrawer = useGetFreight(
    getFreightQuery.data?.data.groups?.['']?.items || [],
    freightName
  );

  useSyncFreight({
    project: name,
    freights: freightById,
    freightInStages
  });

  const freights = getFreightQuery.data?.data?.groups?.['']?.items || [];

  usePersistPreferredFilter(projectName || '', preferredFilter);

  useWatchFreight(projectName || '', preferredFilter.warehouses, !getFreightQuery.isLoading);

  if (loading) {
    return <LoadingState />;
  }

  if (projectQuery.error) {
    return (
      <Result
        status='404'
        title='Error'
        subTitle={projectQuery.error?.message}
        extra={
          <Button type='primary' onClick={() => navigate(paths.projects)}>
            Go to Projects Page
          </Button>
        }
      />
    );
  }

  return (
    <ActionContext.Provider value={action}>
      <FreightTimelineControllerContext.Provider
        value={{
          viewingFreight,
          setPreferredFilter,
          preferredFilter,
          setViewingFreight,
          stageSearch,
          setStageSearch
        }}
      >
        <DictionaryContext.Provider
          value={{
            freightInStages,
            freightById,
            stageAutoPromotionMap,
            subscribersByStage,
            stageByName,
            argocdShards,
            heartbeatsByController,
            defaultControllerName,
            hasAnalysisRunLogsUrlTemplate: getConfigQuery?.data?.data?.hasAnalysisRunLogsUrlTemplate
          }}
        >
          <ColorContext.Provider value={{ stageColorMap, warehouseColorMap }}>
            <DndPromotionContext projectName={projectName || ''}>
              <div className='overflow-hidden h-full flex flex-col'>
                <FreightTimeline freights={freights} project={projectName || ''} />

                <div className='w-full flex-1 relative overflow-auto'>
                  <Flex
                    gap={12}
                    className={classNames(
                      'top-2 right-2 left-2',
                      pipelineView === 'graph' ? 'absolute' : 'pt-2 px-2'
                    )}
                    align='flex-start'
                  >
                    <GraphFilters
                      warehouses={warehouses}
                      stages={stages}
                      freights={freights}
                      pipelineView={pipelineView}
                      setPipelineView={setPipelineView}
                      className='z-10'
                    />
                    <Dropdown
                      className='ml-auto z-10'
                      trigger={['click']}
                      menu={{
                        items: [
                          {
                            key: '0',
                            label: (
                              <Link
                                to={generatePath(paths.createStage, {
                                  name: project.metadata?.name
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
                                  name: project.metadata?.name
                                })}
                              >
                                Warehouse
                              </Link>
                            )
                          },
                          {
                            key: '2',
                            label: 'Freight',
                            children: warehouses.map((warehouse) => ({
                              key: warehouse?.metadata?.name || '',
                              label: warehouse?.metadata?.name || '',
                              onClick: () => {
                                navigate(
                                  generatePath(paths.warehouse, {
                                    name: project.metadata?.name,
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
                    <Button
                      className='z-10'
                      icon={<FontAwesomeIcon icon={faDocker} />}
                      onClick={() =>
                        setPreferredFilter({
                          ...preferredFilter,
                          images: !preferredFilter?.images
                        })
                      }
                    />
                  </Flex>
                  {preferredFilter?.images && (
                    <div className='w-[450px] absolute right-2 top-20 z-10'>
                      <Images
                        hide={() =>
                          setPreferredFilter({
                            ...preferredFilter,
                            images: !preferredFilter?.images
                          })
                        }
                        images={listImagesQuery.data?.data || {}}
                        project={projectName || ''}
                        stages={stages}
                        warehouses={warehouses}
                      />
                    </div>
                  )}
                  {listStagesQuery.isLoading && (
                    <div className='mt-20'>
                      <LoadingState />
                    </div>
                  )}
                  {pipelineView === 'graph' && stages && (
                    <Graph
                      project={project.metadata?.name || ''}
                      warehouses={warehouses}
                      stages={stages}
                    />
                  )}
                  {pipelineView === 'list' && stages && (
                    <PipelineListView
                      stages={stages}
                      warehouses={warehouses}
                      project={projectName || ''}
                      freights={freights}
                      className='mt-2'
                    />
                  )}
                </div>
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
                  project={project?.metadata?.name}
                  warehouses={mapToNames(warehouses)}
                  stages={stages}
                />
              )}

              <CreateWarehouse
                visible={Boolean(props.creatingWarehouse)}
                hide={() =>
                  navigate(generatePath(paths.project, { name: project?.metadata?.name }))
                }
              />
            </DndPromotionContext>
          </ColorContext.Provider>
        </DictionaryContext.Provider>
      </FreightTimelineControllerContext.Provider>
    </ActionContext.Provider>
  );
};
