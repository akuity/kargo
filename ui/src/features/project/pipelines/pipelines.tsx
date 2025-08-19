import { useQuery } from '@connectrpc/connect-query';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex, Result } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { generatePath, Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';

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
  getProject,
  listImages,
  listStages,
  listWarehouses,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { FreightList } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Project } from '@ui/gen/api/v1alpha1/generated_pb';

import { ActionContext } from './context/action-context';
import { DictionaryContext } from './context/dictionary-context';
import { FreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { FreightTimeline } from './freight/freight-timeline';
import { Graph } from './graph/graph';
import { GraphFilters } from './graph-filters';
import { Images } from './image-history/images';
import { PipelineListView } from './list/list-view';
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
  const [searchParams, setSearchParams] = useSearchParams();

  const projectQuery = useQuery(getProject, { name });

  const project = projectQuery.data?.result?.value as Project;

  const projectName = name;

  const pipelineView: 'graph' | 'list' = (searchParams.get('view') as 'graph' | 'list') || 'graph';
  const setPipelineView = (nextView: 'graph' | 'list') => {
    const currentSearchParams = new URLSearchParams(searchParams);
    currentSearchParams.set('view', nextView);
    setSearchParams(currentSearchParams);
  };

  const listImagesQuery = useQuery(listImages, { project: name });

  const getFreightQuery = useQuery(queryFreight, { project: projectName });

  const listWarehousesQuery = useQuery(listWarehouses, {
    project: projectName
  });

  const listStagesQuery = useQuery(listStages, { project: projectName });

  const loading =
    projectQuery.isLoading ||
    getFreightQuery.isLoading ||
    listWarehousesQuery.isLoading ||
    listStagesQuery.isLoading;

  const promote = freight && stage ? { freight, stage } : undefined;

  const navigate = useNavigate();

  const action = useAction();

  const stageDetails =
    stageName && listStagesQuery.data?.stages.find((s) => s?.metadata?.name === stageName);

  const warehouseColorMap = useMemo(
    () =>
      getColors(
        project?.metadata?.name || '',
        listWarehousesQuery.data?.warehouses || [],
        'warehouses'
      ),
    [project, listWarehousesQuery.data?.warehouses]
  );

  const stageColorMap = useMemo(
    () => getColors(project?.metadata?.name || '', listStagesQuery.data?.stages || []),
    [project, listStagesQuery.data?.stages]
  );

  const [preferredFilter, setPreferredFilter] = useFreightTimelineControllerStore(
    projectName || ''
  );

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const stages = listStagesQuery?.data?.stages || [];
  const freightInStages = useFreightInStage(stages);
  const freightById = useFreightById(getFreightQuery.data?.groups?.['']?.freight || []);
  const stageAutoPromotionMap = useStageAutoPromotionMap(stages);
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

  useSyncFreight({
    freights: freightById,
    freightInStages
  });

  const freights = getFreightQuery.data?.groups?.['']?.freight || [];

  usePersistPreferredFilter(projectName || '', preferredFilter);

  useWatchFreight(projectName || '');

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
            <FreightTimeline freights={freights} project={projectName || ''} />

            <div className='w-full h-full relative'>
              <Flex
                gap={12}
                className={classNames(
                  'top-2 right-2 left-2',
                  pipelineView === 'graph' ? 'absolute' : 'pt-2 px-2'
                )}
                align='flex-start'
              >
                <GraphFilters
                  warehouses={listWarehousesQuery.data?.warehouses || []}
                  stages={listStagesQuery.data?.stages || []}
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
                        children: listWarehousesQuery.data?.warehouses?.map((warehouse) => ({
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
                    images={listImagesQuery.data?.images || {}}
                    project={projectName || ''}
                    stages={listStagesQuery.data?.stages || []}
                    warehouses={listWarehousesQuery.data?.warehouses || []}
                  />
                </div>
              )}
              {pipelineView === 'graph' && (
                <Graph
                  project={project.metadata?.name || ''}
                  warehouses={listWarehousesQuery.data?.warehouses || []}
                  stages={listStagesQuery.data?.stages || []}
                />
              )}
              {pipelineView === 'list' && (
                <PipelineListView
                  stages={listStagesQuery.data?.stages || []}
                  warehouses={listWarehousesQuery.data?.warehouses || []}
                  project={projectName || ''}
                  freights={freights}
                  className='mt-2'
                />
              )}
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
                warehouses={mapToNames(listWarehousesQuery.data?.warehouses || [])}
                stages={listStagesQuery.data?.stages || []}
              />
            )}

            <CreateWarehouse
              visible={Boolean(props.creatingWarehouse)}
              hide={() => navigate(generatePath(paths.project, { name: project?.metadata?.name }))}
            />
          </ColorContext.Provider>
        </DictionaryContext.Provider>
      </FreightTimelineControllerContext.Provider>
    </ActionContext.Provider>
  );
};
