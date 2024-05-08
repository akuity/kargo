import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  faChevronDown,
  faEye,
  faEyeSlash,
  faMasksTheater,
  faPalette,
  faRefresh,
  faWandSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Space, Tooltip, message } from 'antd';
import React, { useMemo } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
import { FreightDetails } from '@ui/features/freight/freight-details';
import { Freightline } from '@ui/features/freightline/freightline';
import { StageDetails } from '@ui/features/stage/stage-details';
import { clearColors } from '@ui/features/stage/utils';
import {
  approveFreight,
  listStages,
  listWarehouses,
  queryFreight,
  refreshWarehouse
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';
import { useDocumentEvent } from '@ui/utils/document';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { CreateStageModal } from './create-stage-modal';
import { CreateWarehouseModal } from './create-warehouse-modal';
import { Images } from './images';
import { RepoNode } from './nodes/repo-node';
import { Nodule, StageNode } from './nodes/stage-node';
import styles from './project-details.module.less';
import { FreightlineAction, NodeType } from './types';
import { LINE_THICKNESS, WAREHOUSE_NODE_HEIGHT } from './utils/graph';
import { isPromoting, usePipelineState } from './utils/state';
import { usePipelineGraph } from './utils/use-pipeline-graph';
import { onError } from './utils/util';
import { Watcher } from './utils/watcher';
import { WarehouseDetails } from './warehouse/warehouse-details';

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

  const [highlightedStages, setHighlightedStages] = React.useState<{ [key: string]: boolean }>({});
  const [hideSubscriptions, setHideSubscriptions] = useLocalStorage(
    `${name}-hideSubscriptions`,
    false
  );

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

  const [nodes, connectors, box, sortedStages, stageColorMap] = usePipelineGraph(
    name,
    data?.stages || [],
    warehouseData?.warehouses || [],
    hideSubscriptions
  );

  const [fullFreightById, setFullFreightById] = React.useState<{ [key: string]: Freight }>({});

  const { mutate: manualApproveAction } = useMutation(approveFreight, {
    onError,
    onSuccess: () => {
      message.success(`Freight ${state.freight} has been manually approved.`);
      state.clear();
    }
  });

  const [stagesPerFreight, subscribersByStage] = useMemo(() => {
    const stagesPerFreight: { [key: string]: Stage[] } = {};
    const subscribersByStage = {} as { [key: string]: Stage[] };
    (data?.stages || []).forEach((stage) => {
      const items = stagesPerFreight[stage.status?.currentFreight?.name || ''] || [];
      stagesPerFreight[stage.status?.currentFreight?.name || ''] = [...items, stage];
      stage?.spec?.subscriptions?.upstreamStages.forEach((item) => {
        const items = subscribersByStage[item.name || ''] || [];
        subscribersByStage[item.name || ''] = [...items, stage];
      });
    });
    return [stagesPerFreight, subscribersByStage];
  }, [data, freightData]);

  React.useEffect(() => {
    const fullFreightById: { [key: string]: Freight } = {};
    (freightData?.groups['']?.freight || []).forEach((freight) => {
      fullFreightById[freight?.metadata?.name || ''] = freight;
    });
    setFullFreightById(fullFreightById);
  }, [freightData]);

  if (isLoading || isLoadingFreight) return <LoadingState />;

  const stage = stageName && (data?.stages || []).find((item) => item.metadata?.name === stageName);
  const freight =
    freightName &&
    (freightData?.groups['']?.freight || []).find((item) => item.metadata?.name === freightName);
  const warehouse =
    warehouseName &&
    (warehouseData?.warehouses || []).find((item) => item.metadata?.name === warehouseName);

  const isFaded = (stage: Stage): boolean => {
    if (!isPromoting(state)) {
      return false;
    }
    if (state.action === 'promote') {
      return state.stage !== stage?.metadata?.name;
    }
    if (state.action === 'promoteSubscribers') {
      return !subscribersByStage[state.stage || '']?.find(
        (item) => item.metadata?.name === stage?.metadata?.name
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
      <ColorContext.Provider value={stageColorMap}>
        <Freightline
          highlightedStages={
            state.action === FreightlineAction.ManualApproval ? {} : highlightedStages
          }
          refetchFreight={refetchFreightData}
          onHover={onHover}
          freight={freightData?.groups['']?.freight || []}
          state={state}
          promotionEligible={{}}
          subscribersByStage={subscribersByStage}
          stagesPerFreight={stagesPerFreight}
        />
        <div className='flex flex-grow w-full'>
          <div className={`overflow-hidden flex-grow w-full h-full ${styles.dag}`}>
            <div className='flex justify-end items-center p-4 mb-4'>
              <div>
                <Tooltip title='Reassign Stage Colors'>
                  <Button
                    type='default'
                    className='mr-2'
                    onClick={() => {
                      clearColors(name || '');
                      window.location.reload();
                    }}
                  >
                    <FontAwesomeIcon icon={faPalette} />
                  </Button>
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
                  <Button type='primary' icon={<FontAwesomeIcon icon={faWandSparkles} size='1x' />}>
                    <Space>
                      Create
                      <FontAwesomeIcon icon={faChevronDown} size='xs' />
                    </Space>
                  </Button>
                </Dropdown>
              </div>
            </div>
            <div className='overflow-auto p-6 h-full'>
              <div
                className='relative'
                style={{ width: box?.width, height: box?.height, margin: '0 auto' }}
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
                          currentFreight={
                            fullFreightById[node.data?.status?.currentFreight?.name || '']
                          }
                          hasNoSubscribers={
                            (subscribersByStage[node?.data?.metadata?.name || ''] || []).length <= 1
                          }
                          onPromoteClick={(type: FreightlineAction) => {
                            if (state.stage === node.data?.metadata?.name) {
                              // deselect
                              state.clear();
                            } else {
                              const stageName = node.data?.metadata?.name || '';
                              // default to current freight when promoting subscribers
                              state.select(
                                type,
                                stageName,
                                type === FreightlineAction.PromoteSubscribers
                                  ? node.data?.status?.currentFreight?.name || ''
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
                            state.action === FreightlineAction.ManualApproval
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
                          approving={state.action === FreightlineAction.ManualApproval}
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
                          <div className='flex w-full h-full'>
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
                              className='m-auto'
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
                      className='absolute bg-gray-400'
                      style={{
                        padding: 0,
                        margin: 0,
                        height: LINE_THICKNESS,
                        width: line.width,
                        left: line.x,
                        top: line.y,
                        transform: `rotate(${line.angle}deg)`
                      }}
                      key={i}
                    />
                  ))
                )}
              </div>
            </div>
          </div>

          <Images project={name as string} stages={sortedStages || []} />
        </div>
        {stage && <StageDetails stage={stage} />}
        {freight && <FreightDetails freight={freight} />}
        {warehouse && <WarehouseDetails warehouse={warehouse} />}
      </ColorContext.Provider>
    </div>
  );
};
