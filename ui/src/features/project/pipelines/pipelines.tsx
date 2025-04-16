import { useQuery } from '@connectrpc/connect-query';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import {
  faChevronDown,
  faMasksTheater,
  faPalette,
  faWandSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Spin, Tooltip } from 'antd';
import classNames from 'classnames';
import React, { Suspense, lazy, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
const FreightDetails = lazy(() => import('@ui/features/freight/freight-details'));
const FreightTimeline = lazy(() => import('@ui/features/freight-timeline/freight-timeline'));
const StageDetails = lazy(() => import('@ui/features/stage/stage-details'));
const CreateStage = lazy(() => import('@ui/features/stage/create-stage'));
const CreateWarehouse = lazy(() => import('@ui/features/stage/create-warehouse/create-warehouse'));
import { SuspenseSpin } from '@ui/features/common/suspense-spin';
import {
  getCurrentFreight,
  getCurrentFreightWarehouse,
  mapToNames
} from '@ui/features/common/utils';
const FreightTimelineHeader = lazy(
  () => import('@ui/features/freight-timeline/freight-timeline-header')
);
import { FreightTimelineWrapper } from '@ui/features/freight-timeline/freight-timeline-wrapper';
import { clearColors, getColors } from '@ui/features/stage/utils';
import { queryCache } from '@ui/features/utils/cache';
import {
  listStages,
  listImages,
  listWarehouses,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Project, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { useDocumentEvent } from '@ui/utils/document';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { PipelineContext } from './context/pipeline-context';
import { Graph } from './graph';
import { Images } from './images';
import styles from './project-details.module.less';
import { CollapseMode, FreightTimelineAction } from './types';
import { usePipelineState } from './utils/state';
import { Watcher } from './utils/watcher';

import '@xyflow/react/dist/style.css';

const WarehouseDetails = lazy(() => import('./warehouse/warehouse-details'));

export const Pipelines = ({
  project,
  creatingStage,
  creatingWarehouse
}: {
  project: Project;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name, stageName, freightName, warehouseName } = useParams();
  const { data, isLoading } = useQuery(listStages, { project: name });
  const {
    data: imageData,
    isLoading: isLoadingImages
    // refetch: refetchListImages
  } = useQuery(listImages, { project: name });

  const navigate = useNavigate();
  const {
    data: freightData,
    isLoading: isLoadingFreight,
    refetch: refetchFreightData
  } = useQuery(queryFreight, { project: name });

  const { data: warehouseData, isLoading: isLoadingWarehouse } = useQuery(listWarehouses, {
    project: name
  });

  const state = usePipelineState();

  const isVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  const [highlightedStages, setHighlightedStages] = React.useState<{ [key: string]: boolean }>({});

  const [selectedWarehouse, setSelectedWarehouse] = React.useState('');
  // remember what user selected explicitly
  const lastExplicitlySelectedWarehouse = useRef('');
  const [freightTimelineCollapsed, setFreightTimelineCollapsed] = React.useState(
    CollapseMode.Expanded
  );

  const [hideImages, setHideImages] = useLocalStorage(
    `${name}-hide-images`,
    Object.keys(imageData?.images || {}).length
  );

  const hideImageSection = useCallback(() => {
    setHideImages(true);
  }, [setHideImages]);

  const [isNew, setIsNew] = useLocalStorage(`${name}-is-new`, false);

  useEffect(() => {
    if (Object.keys(imageData?.images || {}).length > 0 && isNew) {
      setIsNew(false);
      setHideImages(false);
    }
  }, [imageData?.images]);

  const warehouseMap = useMemo(() => {
    const map = {} as { [key: string]: Warehouse };
    (warehouseData?.warehouses || []).forEach((warehouse) => {
      map[warehouse.metadata?.name || ''] = warehouse;
    });
    return map;
  }, [warehouseData]);

  const filteredFreight = useMemo(() => {
    const allFreight = freightData?.groups['']?.freight || [];
    const filteredFreight = [] as Freight[];
    allFreight.forEach((f) => {
      if (
        !selectedWarehouse ||
        (f?.origin?.kind === 'Warehouse' && f?.origin.name === selectedWarehouse)
      ) {
        filteredFreight.push(f);
      }
    });
    return filteredFreight;
  }, [freightData, selectedWarehouse]);

  const autoPromotionMap = useMemo(() => {
    const apMap = {} as { [key: string]: boolean };
    (project?.spec?.promotionPolicies || []).forEach((policy) => {
      if (policy.stage) {
        apMap[policy.stage] = policy.autoPromotionEnabled || false;
      }
    });
    return apMap;
  }, [project]);

  const client = useQueryClient();

  useEffect(() => {
    if (!name || !isVisible) {
      return;
    }

    const watcher = new Watcher(name, client);

    watcher.watchStages(queryCache.imageStageMatrix.update);
    watcher.watchWarehouses({
      refreshHook: refetchFreightData
    });

    return () => {
      watcher.cancelWatch();
    };
  }, [name, client, isVisible]);

  const [stagesPerFreight, subscribersByStage, stagesWithFreight] = useMemo(() => {
    const stagesPerFreight: { [key: string]: Stage[] } = {};
    const subscribersByStage = {} as { [key: string]: Set<string> };
    let stagesWithFreight = 0;
    (data?.stages || []).forEach((stage) => {
      const currentFreight = getCurrentFreight(stage);
      if (currentFreight.length > 0) {
        stagesWithFreight++;
      }
      (currentFreight || []).forEach((f) => {
        if (!stagesPerFreight[f.name || '']) {
          stagesPerFreight[f.name || ''] = [];
        }
        stagesPerFreight[f.name || ''].push(stage);
      });
      stage?.spec?.requestedFreight?.forEach((item) => {
        if (!item.sources?.direct) {
          (item?.sources?.stages || []).forEach((name) => {
            if (!subscribersByStage[name]) {
              subscribersByStage[name] = new Set();
            }
            subscribersByStage[name].add(stage?.metadata?.name || '');
          });
        }
      });
    });
    return [stagesPerFreight, subscribersByStage, stagesWithFreight];
  }, [data, freightData]);

  const fullFreightById: { [key: string]: Freight } = useMemo(() => {
    const freightMap: { [key: string]: Freight } = {};
    (freightData?.groups['']?.freight || []).forEach((freight) => {
      freightMap[freight?.metadata?.name || ''] = freight;
    });
    return freightMap;
  }, [freightData]);

  // if we find any stage with freight and UI don't have details, refresh freights
  useEffect(() => {
    if (fullFreightById && stagesPerFreight) {
      const freights = Object.keys(fullFreightById || {});
      const freightsInStages = Object.keys(stagesPerFreight || {});

      for (const freightInStage of freightsInStages) {
        if (!freights?.find((freight) => freight === freightInStage)) {
          refetchFreightData();
          return;
        }
      }
    }
  }, [stagesPerFreight, fullFreightById]);

  const pipelinesConfigRef = useRef<HTMLDivElement>(null);

  const onHover = useCallback(
    (h: boolean, id: string, isStage?: boolean) => {
      const stages = {} as { [key: string]: boolean };
      if (!h) {
        setHighlightedStages(stages);
        return;
      }
      if (isStage) {
        stages[id] = true;
      } else {
        (stagesPerFreight[id] || []).forEach((stage) => {
          stages[stage.metadata?.name || ''] = true;
        });
      }
      setHighlightedStages(stages);
    },
    [stagesPerFreight]
  );

  const onPromoteClick = useCallback(
    (stage: Stage, type: FreightTimelineAction) => {
      // which warehouse to select?
      // check if they have filter applied in freight timeline
      // if not, then select the warehouse of latest promoted freight
      if (selectedWarehouse === '') {
        setSelectedWarehouse(getCurrentFreightWarehouse(stage));
      }

      if (state.stage === stage?.metadata?.name) {
        // deselect
        state.clear();

        setSelectedWarehouse(lastExplicitlySelectedWarehouse.current);
      } else {
        const stageName = stage?.metadata?.name || '';
        state.select(type, stageName, undefined);
      }
    },
    [state]
  );

  const warehouseColorMap = useMemo(
    () => getColors(name || '', warehouseData?.warehouses || [], 'warehouses'),
    [name, warehouseData?.warehouses]
  );

  const stageColorMap = useMemo(
    () => getColors(name || '', data?.stages || []),
    [name, data?.stages]
  );

  const [hideParents, setHideParents] = useState<string[]>([]);

  if (isLoading || isLoadingFreight || isLoadingImages || isLoadingWarehouse)
    return <LoadingState />;

  const stage = stageName && (data?.stages || []).find((item) => item.metadata?.name === stageName);
  const freight = freightName && fullFreightById[freightName];
  const warehouse = warehouseName && warehouseMap[warehouseName];

  return (
    <div className='flex flex-col flex-grow'>
      <ColorContext.Provider value={{ stageColorMap, warehouseColorMap }}>
        <div className='bg-gray-100'>
          <FreightTimelineHeader
            promotingStage={state.stage}
            action={state.action}
            cancel={() => {
              state.clear();
              setSelectedWarehouse('');
            }}
            downstreamSubs={Array.from(subscribersByStage[state.stage || ''] || [])}
            selectedWarehouse={selectedWarehouse || ''}
            setSelectedWarehouse={(explicitlySelectedWarehouse) => {
              lastExplicitlySelectedWarehouse.current = explicitlySelectedWarehouse;
              setSelectedWarehouse(explicitlySelectedWarehouse);
            }}
            warehouses={warehouseMap}
            collapsed={freightTimelineCollapsed}
            setCollapsed={setFreightTimelineCollapsed}
            collapsable={
              Object.keys(stagesPerFreight).reduce(
                (acc, cur) => (cur?.length > 0 ? acc + stagesPerFreight[cur].length : acc),
                0
              ) > 0
            }
          />
          <FreightTimelineWrapper>
            <Suspense
              fallback={
                <div className='h-full w-full flex items-center justify-center'>
                  <Spin />
                </div>
              }
            >
              <FreightTimeline
                highlightedStages={
                  state.action === FreightTimelineAction.ManualApproval ? {} : highlightedStages
                }
                refetchFreight={refetchFreightData}
                onHover={onHover}
                freight={filteredFreight}
                state={state}
                promotionEligible={{}}
                stagesPerFreight={stagesPerFreight}
                collapsed={freightTimelineCollapsed}
                setCollapsed={setFreightTimelineCollapsed}
                stageCount={stagesWithFreight}
              />
            </Suspense>
          </FreightTimelineWrapper>
        </div>
        {/* TODO: Use original canvas approach for greater performance, flexibility and pixel perfect */}
        <div className={styles.dag}>
          <div className={styles.staticView} ref={pipelinesConfigRef}>
            <div className={styles.pipelinesViewConfig}>
              <div className={styles.toolbar}>
                <Tooltip title='Regenerate Stage Colors'>
                  <Button
                    type='dashed'
                    onClick={() => {
                      clearColors(name || '');
                      clearColors(name || '', 'warehouses');
                      window.location.reload();
                    }}
                    icon={<FontAwesomeIcon icon={faPalette} />}
                  />
                </Tooltip>{' '}
                <Dropdown
                  menu={{
                    items: [
                      {
                        key: '1',
                        label: (
                          <>
                            <FontAwesomeIcon icon={faMasksTheater} size='xs' className='mr-2' />{' '}
                            Create Stage
                          </>
                        ),
                        onClick: () => navigate(generatePath(paths.createStage, { name }))
                      },
                      {
                        key: '2',
                        label: (
                          <>
                            <FontAwesomeIcon icon={faWarehouse} size='xs' className='mr-2' /> Create
                            Warehouse
                          </>
                        ),
                        onClick: () => navigate(generatePath(paths.createWarehouse, { name }))
                      }
                    ]
                  }}
                  placement='bottomRight'
                  trigger={['click']}
                >
                  <Button icon={<FontAwesomeIcon icon={faWandSparkles} size='1x' />}>
                    <FontAwesomeIcon icon={faChevronDown} size='xs' className='-mr-2' />
                  </Button>
                </Dropdown>
                {!!hideImages && (
                  <Tooltip title='Show Images'>
                    <Button
                      icon={<FontAwesomeIcon icon={faDocker} />}
                      onClick={() => setHideImages(false)}
                      className='ml-2'
                      type='dashed'
                    />
                  </Tooltip>
                )}
              </div>
            </div>

            <div className={classNames(styles.imagesMatrix, { hidden: hideImages })}>
              <Images
                project={name as string}
                // TODO(Marvin9): does sortedStages matter?
                stages={data?.stages || []}
                hide={hideImageSection}
                images={imageData?.images || {}}
              />
            </div>
          </div>

          <PipelineContext.Provider
            value={{
              state,
              autoPromotionMap,
              highlightedStages,
              fullFreightById,
              subscribersByStage,
              selectedWarehouse,
              project: project?.metadata?.name || '',
              onHover,
              onPromoteClick,
              setSelectedWarehouse(newWarehouse) {
                lastExplicitlySelectedWarehouse.current = newWarehouse;
                setSelectedWarehouse(newWarehouse);
              },
              hideParents,
              onHideParents: setHideParents
            }}
          >
            <Graph
              project={name || ''}
              stages={data?.stages || []}
              warehouses={warehouseData?.warehouses || []}
            />
          </PipelineContext.Provider>
        </div>
        <SuspenseSpin>
          {stage && <StageDetails stage={stage} />}
          {freight && <FreightDetails freight={freight} refetchFreight={refetchFreightData} />}
          {warehouse && (
            <WarehouseDetails warehouse={warehouse} refetchFreight={() => refetchFreightData()} />
          )}
          {creatingStage && (
            <CreateStage
              project={name}
              warehouses={mapToNames(warehouseData?.warehouses || [])}
              stages={mapToNames(data?.stages || [])}
            />
          )}
          <CreateWarehouse
            visible={Boolean(creatingWarehouse)}
            hide={() => navigate(generatePath(paths.project, { name }))}
          />
        </SuspenseSpin>
      </ColorContext.Provider>
    </div>
  );
};
