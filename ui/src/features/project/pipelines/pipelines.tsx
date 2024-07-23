import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import {
  faChevronDown,
  faExpand,
  faEye,
  faEyeSlash,
  faFilter,
  faMagnifyingGlassMinus,
  faMagnifyingGlassPlus,
  faMasksTheater,
  faPalette,
  faRefresh,
  faWandSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Spin, Tooltip, message } from 'antd';
import React, { Suspense, lazy, useMemo } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
const FreightDetails = lazy(() => import('@ui/features/freight/freight-details'));
const FreightTimeline = lazy(() => import('@ui/features/freight-timeline/freight-timeline'));
const StageDetails = lazy(() => import('@ui/features/stage/stage-details'));
import { SuspenseSpin } from '@ui/features/common/suspense-spin';
import { getCurrentFreight } from '@ui/features/common/utils';
import { FreightTimelineHeader } from '@ui/features/freight-timeline/freight-timeline-header';
import { FreightTimelineWrapper } from '@ui/features/freight-timeline/freight-timeline-wrapper';
import { clearColors } from '@ui/features/stage/utils';
import {
  approveFreight,
  listStages,
  listWarehouses,
  queryFreight,
  refreshWarehouse
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';
import { useDocumentEvent } from '@ui/utils/document';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import CreateStageModal from './create-stage-modal';
import CreateWarehouseModal from './create-warehouse-modal';
import { Images } from './images';
import { RepoNode } from './nodes/repo-node';
import { Nodule, StageNode } from './nodes/stage-node';
import styles from './project-details.module.less';
import { FreightTimelineAction, NodeType } from './types';
import { LINE_THICKNESS, WAREHOUSE_NODE_HEIGHT } from './utils/graph';
import { isPromoting, usePipelineState } from './utils/state';
import { usePipelineGraph } from './utils/use-pipeline-graph';
import { onError } from './utils/util';
import { Watcher } from './utils/watcher';

const WarehouseDetails = lazy(() => import('./warehouse/warehouse-details'));

export const Pipelines = () => {
  const { name, stageName, freightName, warehouseName } = useParams();
  const { data, isLoading } = useQuery(listStages, { project: name });
  const navigate = useNavigate();
  const {
    data: freightData,
    isLoading: isLoadingFreight,
    refetch: refetchFreightData
  } = useQuery(queryFreight, { project: name });

  const { data: warehouseData } = useQuery(listWarehouses, {
    project: name
  });

  const { show: showCreateStage } = useModal(
    name ? (p) => <CreateStageModal {...p} project={name} /> : undefined
  );
  const { show: showCreateWarehouse } = useModal(
    name ? (p) => <CreateWarehouseModal {...p} project={name} /> : undefined
  );

  const state = usePipelineState();

  const isVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  const { mutate: refreshWarehouseAction } = useMutation(refreshWarehouse, {
    onError,
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
      state.clear();
      refetchFreightData();
    }
  });

  const [zoom, setZoom] = React.useState(100);

  const [highlightedStages, setHighlightedStages] = React.useState<{ [key: string]: boolean }>({});
  const [hideSubscriptions, setHideSubscriptions] = useLocalStorage(
    `${name}-hideSubscriptions`,
    false
  );

  const [selectedWarehouse, setSelectedWarehouse] = React.useState('');
  const [freightTimelineCollapsed, setFreightTimelineCollapsed] = React.useState(false);

  const [hideImages, setHideImages] = useLocalStorage(`${name}-hideImages`, false);

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
        f.warehouse === selectedWarehouse ||
        (f?.origin?.kind === 'Warehouse' && f?.origin.name === selectedWarehouse)
      ) {
        filteredFreight.push(f);
      }
    });
    return filteredFreight;
  }, [freightData, selectedWarehouse]);

  const client = useQueryClient();

  React.useEffect(() => {
    if (!data || !isVisible || !warehouseData || !name) {
      return;
    }

    const watcher = new Watcher(name, client);
    watcher.watchStages(data.stages.slice());
    watcher.watchWarehouses(warehouseData?.warehouses || [], refetchFreightData);

    return () => watcher.cancelWatch();
  }, [isLoading, isVisible, name]);

  const [nodes, connectors, box, sortedStages, stageColorMap, warehouseColorMap] = usePipelineGraph(
    name,
    data?.stages || [],
    warehouseData?.warehouses || [],
    hideSubscriptions
  );

  const { mutate: manualApproveAction } = useMutation(approveFreight, {
    onError,
    onSuccess: () => {
      message.success(`Freight ${state.freight} has been manually approved.`);
      refetchFreightData();
      state.clear();
    }
  });

  const [stagesPerFreight, subscribersByStage] = useMemo(() => {
    const stagesPerFreight: { [key: string]: Stage[] } = {};
    const subscribersByStage = {} as { [key: string]: Set<string> };
    (data?.stages || []).forEach((stage) => {
      const items = stagesPerFreight[stage.status?.currentFreight?.name || ''] || [];
      stagesPerFreight[stage.status?.currentFreight?.name || ''] = [...items, stage];
      stage?.spec?.subscriptions?.upstreamStages.forEach((item) => {
        if (!subscribersByStage[item.name || '']) {
          subscribersByStage[item.name || ''] = new Set();
        }
        subscribersByStage[item.name || ''].add(stage?.metadata?.name || '');
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
    return [stagesPerFreight, subscribersByStage];
  }, [data, freightData]);

  const fullFreightById: { [key: string]: Freight } = useMemo(() => {
    const freightMap: { [key: string]: Freight } = {};
    (freightData?.groups['']?.freight || []).forEach((freight) => {
      freightMap[freight?.metadata?.name || ''] = freight;
    });
    return freightMap;
  }, [freightData]);

  if (isLoading || isLoadingFreight) return <LoadingState />;

  const stage = stageName && (data?.stages || []).find((item) => item.metadata?.name === stageName);
  const freight = freightName && fullFreightById[freightName];
  const warehouse = warehouseName && warehouseMap[warehouseName];

  const isFaded = (stage: Stage): boolean => {
    if (!isPromoting(state)) {
      return false;
    }
    if (state.action === 'promote') {
      return state.stage !== stage?.metadata?.name;
    }
    if (state.action === 'promoteSubscribers') {
      return (
        !stage?.metadata?.name || !subscribersByStage[state.stage || '']?.has(stage.metadata.name)
      );
    }
    return false;
  };

  const onHover = (h: boolean, id: string, isStage?: boolean) => {
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
  };

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
            setSelectedWarehouse={setSelectedWarehouse}
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
              />
            </Suspense>
          </FreightTimelineWrapper>
        </div>
        <div className={`flex flex-grow w-full ${styles.dag}`}>
          <div className={`overflow-hidden flex-grow w-full h-full`}>
            <div className='flex justify-end items-center p-4 mb-4'>
              <div className='flex gap-2'>
                {zoom !== 100 && (
                  <Button onClick={() => setZoom(100)} icon={<FontAwesomeIcon icon={faExpand} />} />
                )}
                <Button
                  onClick={() => setZoom((prev) => Math.max(10, prev - 10))}
                  icon={<FontAwesomeIcon icon={faMagnifyingGlassMinus} />}
                />
                <Button
                  onClick={() => setZoom((prev) => Math.min(200, prev + 10))}
                  icon={<FontAwesomeIcon icon={faMagnifyingGlassPlus} />}
                />
                <Tooltip title='Reassign Stage Colors'>
                  <Button
                    type='default'
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
                            Stage
                          </>
                        ),
                        onClick: () => showCreateStage()
                      },
                      {
                        key: '2',
                        label: (
                          <>
                            <FontAwesomeIcon icon={faWarehouse} size='xs' className='mr-2' />{' '}
                            Warehouse
                          </>
                        ),
                        onClick: () => showCreateWarehouse()
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
                {hideImages && (
                  <Tooltip title='Show Images'>
                    <Button
                      icon={<FontAwesomeIcon icon={faDocker} />}
                      onClick={() => setHideImages(false)}
                      className='ml-2'
                    />
                  </Tooltip>
                )}
              </div>
            </div>
            <div className='overflow-auto p-6 h-full'>
              <div
                className='relative'
                style={{
                  width: box?.width,
                  height: box?.height,
                  margin: '0 auto',
                  zoom: `${zoom}%`
                }}
              >
                {nodes?.map((node, index) => (
                  <div
                    key={index}
                    className='absolute'
                    style={{
                      ...node,
                      color: 'inherit'
                    }}
                  >
                    {node.type === NodeType.STAGE ? (
                      <>
                        <StageNode
                          stage={node.data}
                          color={node.color}
                          height={node.height}
                          projectName={name}
                          faded={isFaded(node.data)}
                          currentFreight={getCurrentFreight(node.data).map(
                            (f) => fullFreightById[f.name || '']
                          )}
                          hasNoSubscribers={
                            Array.from(subscribersByStage[node?.data?.metadata?.name || ''] || [])
                              .length <= 1
                          }
                          onPromoteClick={(type: FreightTimelineAction) => {
                            const currentFreight = getCurrentFreight(node.data);
                            const isWarehouseKind = currentFreight.reduce(
                              (acc, cur) => acc || cur?.origin?.kind === 'Warehouse',
                              false
                            );
                            let currentWarehouse = currentFreight[0]?.warehouse || '';
                            if (currentWarehouse === '' && isWarehouseKind) {
                              currentWarehouse =
                                currentFreight[0]?.origin?.name ||
                                node.data?.spec?.subscriptions?.warehouse ||
                                node.data?.spec?.requestedFreight[0]?.origin?.name ||
                                '';
                            }
                            setSelectedWarehouse(currentWarehouse);
                            if (state.stage === node.data?.metadata?.name) {
                              // deselect
                              state.clear();
                              setSelectedWarehouse('');
                            } else {
                              const stageName = node.data?.metadata?.name || '';
                              // default to current freight when promoting subscribers
                              state.select(
                                type,
                                stageName,
                                type === FreightTimelineAction.PromoteSubscribers
                                  ? currentFreight[0].name
                                  : undefined
                              );
                            }
                          }}
                          action={
                            (isPromoting(state) && state.stage === node.data?.metadata?.name) || ''
                              ? state.action
                              : undefined
                          }
                          onClick={
                            state.action === FreightTimelineAction.ManualApproval
                              ? () => {
                                  manualApproveAction({
                                    stage: node.data?.metadata?.name,
                                    project: name,
                                    name: state.freight
                                  });
                                }
                              : undefined
                          }
                          onHover={(h) => onHover(h, node.data?.metadata?.name || '', true)}
                          approving={state.action === FreightTimelineAction.ManualApproval}
                          highlighted={highlightedStages[node.data?.metadata?.name || '']}
                        />
                      </>
                    ) : (
                      <RepoNode
                        nodeData={node}
                        onClick={
                          node.type === NodeType.WAREHOUSE
                            ? () =>
                                navigate(
                                  generatePath(paths.warehouse, {
                                    name,
                                    warehouseName: node.warehouseName
                                  })
                                )
                            : undefined
                        }
                      >
                        {node.type === NodeType.WAREHOUSE && (
                          <div className='flex w-full h-full gap-2 justify-center items-center'>
                            {(Object.keys(warehouseMap) || []).length > 1 && (
                              <Button
                                icon={<FontAwesomeIcon icon={faFilter} />}
                                size='small'
                                type={
                                  selectedWarehouse === node.warehouseName ? 'primary' : 'default'
                                }
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setSelectedWarehouse(
                                    selectedWarehouse === node.warehouseName
                                      ? ''
                                      : node.warehouseName
                                  );
                                }}
                              />
                            )}
                            <Button
                              onClick={(e) => {
                                e.stopPropagation();
                                refreshWarehouseAction({
                                  name: node.warehouseName,
                                  project: name
                                });
                              }}
                              icon={<FontAwesomeIcon icon={faRefresh} />}
                              size='small'
                            >
                              Refresh
                            </Button>
                          </div>
                        )}
                        {node.type === NodeType.WAREHOUSE && (
                          <Nodule
                            nodeHeight={WAREHOUSE_NODE_HEIGHT}
                            onClick={() => setHideSubscriptions(!hideSubscriptions)}
                            icon={hideSubscriptions ? faEye : faEyeSlash}
                            begin={true}
                          />
                        )}
                      </RepoNode>
                    )}
                  </div>
                ))}
                {connectors?.map((connector) =>
                  connector.map((line, i) => (
                    <div
                      className='absolute bg-gray-300 rounded-full'
                      style={{
                        padding: 0,
                        margin: 0,
                        height: LINE_THICKNESS,
                        width: line.width,
                        left: line.x,
                        top: line.y,
                        transform: `rotate(${line.angle}deg)`,
                        backgroundColor: line.color
                      }}
                      key={i}
                    />
                  ))
                )}
              </div>
            </div>
          </div>

          {!hideImages && (
            <div
              className='p-6 pt-4 h-full'
              style={{
                width: '450px'
              }}
            >
              <Images
                project={name as string}
                stages={sortedStages || []}
                hide={() => setHideImages(true)}
              />
            </div>
          )}
        </div>
        <SuspenseSpin>
          {stage && <StageDetails stage={stage} />}
          {freight && <FreightDetails freight={freight} refetchFreight={refetchFreightData} />}
          {warehouse && (
            <WarehouseDetails warehouse={warehouse} refetchFreight={() => refetchFreightData()} />
          )}
        </SuspenseSpin>
      </ColorContext.Provider>
    </div>
  );
};
