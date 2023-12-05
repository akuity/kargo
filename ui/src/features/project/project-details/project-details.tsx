import { createPromiseClient } from '@bufbuild/connect';
import { Timestamp } from '@bufbuild/protobuf';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import {
  faCircleCheck,
  faDiagramProject,
  faEllipsisV,
  faRefresh,
  faTimeline
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Empty, message } from 'antd';
import { graphlib, layout } from 'dagre';
import React, { useState } from 'react';
import { useParams } from 'react-router-dom';

import { transport } from '@ui/config/transport';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { ConfirmPromotionDialogue } from '@ui/features/freightline/confirm-promotion-dialogue';
import { FreightContents } from '@ui/features/freightline/freight-contents';
import { FreightItem, FreightMode } from '@ui/features/freightline/freight-item';
import {
  Freightline,
  FreightlineHeader,
  PromotionType
} from '@ui/features/freightline/freightline';
import { PromotingStageBanner } from '@ui/features/freightline/promoting-stage-banner';
import { StageIndicators } from '@ui/features/freightline/stage-indicators';
import { StageDetails } from '@ui/features/stage/stage-details';
import { getStageColors } from '@ui/features/stage/utils';
import {
  approveFreight,
  getStage,
  listStages,
  listWarehouses,
  promoteStage,
  promoteSubscribers,
  queryFreight,
  refreshWarehouse
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { Freight, Stage, Warehouse } from '@ui/gen/v1alpha1/types_pb';
import { useDocumentEvent } from '@ui/utils/document';

import { Images } from './images';
import { RepoNode } from './nodes/repo-node';
import { StageNode } from './nodes/stage-node';
import styles from './project-details.module.less';
import { NodeType, NodesItemType } from './types';

const lineThickness = 2;
const nodeWidth = 150;
const nodeHeight = 118;

const warehouseNodeWidth = 180;
const warehouseNodeHeight = 140;

const getSeconds = (ts?: Timestamp): number => Number(ts?.seconds) || 0;

export const ProjectDetails = () => {
  const { name, stageName } = useParams();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));
  const { data: freightData, isLoading: isLoadingFreight } = useQuery(
    queryFreight.useQuery({ project: name })
  );

  const [warehouseMap, setWarehouseMap] = useState<{ [key: string]: Warehouse }>({});
  const { data: warehouseData, isLoading: isLoadingWarehouses } = useQuery(
    listWarehouses.useQuery({ project: name })
  );

  const client = useQueryClient();

  const isVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  const { mutate: refreshWarehouseAction } = useMutation({
    ...refreshWarehouse.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
      setPromotingStage(undefined);
    }
  });

  React.useEffect(() => {
    if (!data || !isVisible) {
      return;
    }

    const cancel = new AbortController();

    const watchStages = async () => {
      const promiseClient = createPromiseClient(KargoService, transport);
      const stream = promiseClient.watchStages({ project: name }, { signal: cancel.signal });
      let stages = data.stages.slice();

      for await (const e of stream) {
        const index = stages.findIndex((item) => item.metadata?.name === e.stage?.metadata?.name);
        if (e.type === 'DELETED') {
          if (index !== -1) {
            stages = [...stages.slice(0, index), ...stages.slice(index + 1)];
          }
        } else {
          if (index === -1) {
            stages = [...stages, e.stage as Stage];
          } else {
            stages = [...stages.slice(0, index), e.stage as Stage, ...stages.slice(index + 1)];
          }
        }

        // Update Stages list
        const listStagesQueryKey = listStages.getQueryKey({ project: name });
        client.setQueryData(listStagesQueryKey, { stages });

        // Update Stage details
        const getStageQueryKey = getStage.getQueryKey({
          project: name,
          name: e.stage?.metadata?.name
        });
        client.setQueryData(getStageQueryKey, { stage: e.stage });
      }
    };
    watchStages();

    return () => cancel.abort();
  }, [isLoading, isVisible, name]);

  React.useEffect(() => {
    if (!isLoadingWarehouses) {
      const wm = {} as { [key: string]: Warehouse };
      (warehouseData?.warehouses || []).forEach((w: Warehouse) => {
        wm[w?.metadata?.name || ''] = w;
      });
      setWarehouseMap(wm);
    }
  }, [warehouseData, isLoadingWarehouses]);

  const [stageColorMap, setStageColorMap] = React.useState<{ [key: string]: string }>({});

  const [nodes, connectors, box, sortedStages] = React.useMemo(() => {
    if (!data) {
      return [[], []];
    }

    const g = new graphlib.Graph();
    g.setGraph({ rankdir: 'LR' });
    g.setDefaultEdgeLabel(() => ({}));

    const myNodes = data.stages
      .slice()
      .sort((a, b) => a.metadata?.name?.localeCompare(b.metadata?.name || '') || 0)
      .flatMap((stage) => {
        const n = [
          {
            data: stage,
            type: NodeType.STAGE,
            color: '#000'
          }
        ] as NodesItemType[];

        const warehouseName = stage.spec?.subscriptions?.warehouse;
        if (warehouseName) {
          const cur = warehouseMap[warehouseName];
          cur?.spec?.subscriptions?.forEach((sub) => {
            const type = sub.chart
              ? NodeType.REPO_CHART
              : sub.image
                ? NodeType.REPO_IMAGE
                : NodeType.REPO_GIT;
            n.push({
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              data: sub.chart || sub.image || sub.git || ({} as any),
              stageName: stage.metadata?.name || '',
              warehouseName: cur.metadata?.name || '',
              type
            });
          });
        }

        return n;
      });

    myNodes.forEach((item, index) => {
      if (item.type === NodeType.STAGE) {
        g.setNode(String(index), {
          width: nodeWidth,
          height: nodeHeight
        });
        item.data?.spec?.subscriptions?.upstreamStages.forEach((upstramStage) => {
          const subsIndex = myNodes.findIndex((node) => {
            return node.type === NodeType.STAGE && node.data.metadata?.name === upstramStage.name;
          });

          g.setEdge(String(subsIndex), String(index));
        });
      } else {
        g.setNode(String(index), {
          width: warehouseNodeWidth,
          height: warehouseNodeHeight
        });
        const subsIndex = myNodes.findIndex((node) => {
          return node.type === NodeType.STAGE && node.data.metadata?.name === item.stageName;
        });

        g.setEdge(String(index), String(subsIndex));
      }
    });

    layout(g, { lablepos: 'c' });

    const nodes = myNodes.map((node, index) => {
      const gNode = g.node(String(index));

      return {
        ...node,
        left: gNode.x - gNode.width / 2,
        top: gNode.y - gNode.height / 2,
        width: gNode.width,
        height: gNode.height
      };
    });

    const connectors = g.edges().map((item) => {
      const edge = g.edge(item);
      const points = edge.points;
      if (points.length > 0) {
        // replace first point with the right side of the upstream node
        const upstreamNode = g.node(item.v);
        if (upstreamNode) {
          points[0] = { x: upstreamNode.x + upstreamNode.width / 2, y: upstreamNode.y };
        }
      }
      if (points.length > 1) {
        // replace last point with the right side of the downstream node
        const upstreamNode = g.node(item.w);
        if (upstreamNode) {
          points[points.length - 1] = {
            x: upstreamNode.x - upstreamNode.width / 2,
            y: upstreamNode.y
          };
        }
      }

      const lines = new Array<{ x: number; y: number; width: number; angle: number }>();
      for (let i = 0; i < points.length - 1; i++) {
        const start = points[i];
        const end = points[i + 1];
        const x1 = start.x;
        const y1 = start.y;
        const x2 = end.x;
        const y2 = end.y;

        const width = Math.sqrt((x2 - x1) * (x2 - x1) + (y2 - y1) * (y2 - y1));
        // center
        const cx = (x1 + x2) / 2 - width / 2;
        const cy = (y1 + y2) / 2 - lineThickness / 2;

        const angle = Math.atan2(y1 - y2, x1 - x2) * (180 / Math.PI);
        lines.push({ x: cx, y: cy, width, angle });
      }
      return lines;
    });

    const box = nodes.reduce(
      (acc, node) => ({
        width: Math.max(acc.width, node.left + node.width),
        height: Math.max(acc.height, node.top + node.height)
      }),
      {
        width: 0,
        height: 0
      }
    );

    const sortedStages = nodes
      .filter((item) => item.type === NodeType.STAGE)
      .sort((a, b) => a.left - b.left)
      .map((item) => item.data) as Stage[];

    const scm = getStageColors(name || '', sortedStages);
    setStageColorMap(scm);
    nodes.forEach((node) => {
      if (node.type === NodeType.STAGE) {
        const color = stageColorMap[node.data?.metadata?.uid || ''];
        if (color) {
          node.color = color;
        }
      }
    });

    return [nodes, connectors, box, sortedStages];
  }, [data]);

  const [stagesPerFreight, setStagesPerFreight] = React.useState<{ [key: string]: Stage[] }>({});
  const [promotingStage, setPromotingStage] = React.useState<Stage | undefined>();
  const [promotionType, setPromotionType] = React.useState('default' as PromotionType);
  const [confirmingPromotion, setConfirmingPromotion] = React.useState<string | undefined>();
  const [subscribersByStage, setSubscribersByStage] = React.useState<{ [key: string]: Stage[] }>(
    {}
  );
  const [fullFreightById, setFullFreightById] = React.useState<{ [key: string]: Freight }>({});
  const [promotionEligible, setPromotionEligible] = React.useState<{ [key: string]: boolean }>({});
  const [manuallyApproving, setManuallyApproving] = React.useState<string | undefined>();

  const { mutate: manualApproveAction } = useMutation({
    ...approveFreight.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`Freight ${confirmingPromotion} has been manually approved.`);
      setManuallyApproving(undefined);
    }
  });

  const { mutate: promoteSubscribersAction } = useMutation({
    ...promoteSubscribers.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `All subscribers of "${promotingStage?.metadata?.name}" stage have been promoted.`
      );
      setPromotingStage(undefined);
    }
  });

  const { mutate: promoteAction } = useMutation({
    ...promoteStage.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`Stage "${promotingStage?.metadata?.name}" has been promoted.`);
      setPromotingStage(undefined);
    }
  });

  const {
    data: availableFreightData,
    refetch: refetchAvailableFreight,
    isLoading: isLoadingAvailableFreight
  } = useQuery(
    queryFreight.useQuery({ project: name, stage: promotingStage?.metadata?.name || '' })
  );

  const freightModeFor = (freightID: string): FreightMode => {
    if (manuallyApproving) {
      return manuallyApproving === freightID ? FreightMode.Selected : FreightMode.Disabled;
    }

    if (!promotingStage) {
      return FreightMode.Default;
    }

    if (confirmingPromotion === freightID) {
      return FreightMode.Confirming;
    }

    return promotionEligible[freightID] ? FreightMode.Promotable : FreightMode.Disabled;
  };

  // When in promotion mode, create a map of freight eligible for promotion, indexed by Freight ID
  React.useEffect(() => {
    if (!isLoadingAvailableFreight && promotingStage !== undefined) {
      const initFreight = availableFreightData?.groups['']?.freight || [];
      const availableFreight =
        promotionType === 'default'
          ? initFreight
          : // if promoting subscribers, only include freight that has been verified in the promoting stage
            initFreight.filter(
              (f) => !!f?.status?.verifiedIn[promotingStage?.metadata?.name || '']
            );

      const pe: { [key: string]: boolean } = {};
      ((availableFreight as Freight[]) || []).forEach((f: Freight) => {
        pe[f?.id || ''] = true;
      });
      setPromotionEligible(pe);
    }
  }, [availableFreightData]);

  React.useEffect(() => {
    refetchAvailableFreight();
  }, [promotingStage, promotionType, freightData]);

  React.useEffect(() => {
    const stagesPerFreight: { [key: string]: Stage[] } = {};
    (data?.stages || []).forEach((stage) => {
      const items = stagesPerFreight[stage.status?.currentFreight?.id || ''] || [];
      stagesPerFreight[stage.status?.currentFreight?.id || ''] = [...items, stage];
      stage?.spec?.subscriptions?.upstreamStages.forEach((item) => {
        const items = subscribersByStage[item.name || ''] || [];
        subscribersByStage[item.name || ''] = [...items, stage];
      });
    });
    setStagesPerFreight(stagesPerFreight);
    setSubscribersByStage(subscribersByStage);
  }, [data, freightData]);

  React.useEffect(() => {
    const fullFreightById: { [key: string]: Freight } = {};
    (freightData?.groups['']?.freight || []).forEach((freight) => {
      fullFreightById[freight.id || ''] = freight;
    });
    setFullFreightById(fullFreightById);
  }, [freightData]);

  if (isLoading || isLoadingFreight) return <LoadingState />;

  if (!data || data.stages.length === 0) return <Empty />;
  const stage = stageName && data.stages.find((item) => item.metadata?.name === stageName);

  const isFaded = (stage: Stage): boolean => {
    if (!promotingStage || !confirmingPromotion) {
      return false;
    }
    if (promotionType === 'default') {
      return promotingStage?.metadata?.name !== stage?.metadata?.name;
    }
    if (promotionType === 'subscribers') {
      return !subscribersByStage[promotingStage?.metadata?.name || '']?.find(
        (item) => item.metadata?.name === stage?.metadata?.name
      );
    }
    return false;
  };

  return (
    <div className='flex flex-col flex-grow'>
      <ColorContext.Provider value={stageColorMap}>
        <Freightline
          promotingStage={promotingStage}
          setPromotingStage={setPromotingStage}
          promotionType={promotionType}
          header={
            <FreightlineHeader
              banner={
                promotingStage && (
                  <PromotingStageBanner promotionType={promotionType || 'default'} />
                )
              }
            >
              {!promotingStage && !manuallyApproving && (
                <div className='font-semibold'>
                  <FontAwesomeIcon icon={faTimeline} className='mr-2' />
                  FREIGHTLINE
                </div>
              )}
              {promotingStage && (
                <>
                  PROMOTING {promotionType === 'subscribers' ? 'SUBSCRIBERS OF' : 'CURRENT'} STAGE{' '}
                  <div className='font-semibold flex items-center ml-1'>
                    <div
                      className='px-2 rounded text-white ml-2'
                      style={{
                        backgroundColor: stageColorMap[promotingStage?.metadata?.uid || '']
                      }}
                    >
                      {' '}
                      {promotingStage?.metadata?.name?.toUpperCase()}
                    </div>
                  </div>
                  <div className={styles.headerButton} onClick={() => setPromotingStage(undefined)}>
                    CANCEL
                  </div>
                </>
              )}
              {manuallyApproving && (
                <div className='flex items-center w-full'>
                  MANUALLY APPROVING
                  <div
                    className={styles.headerButton}
                    onClick={() => setManuallyApproving(undefined)}
                  >
                    CANCEL
                  </div>
                </div>
              )}
            </FreightlineHeader>
          }
        >
          <>
            {(freightData?.groups['']?.freight || [])
              .sort(
                (a, b) =>
                  getSeconds(b.metadata?.creationTimestamp) -
                  getSeconds(a.metadata?.creationTimestamp)
              )
              .map((f, i) => {
                const id = f?.metadata?.name || `${i}`;
                return (
                  <FreightItem
                    freight={f || undefined}
                    key={id}
                    onClick={() => {
                      if (promotingStage && promotionEligible[id]) {
                        setConfirmingPromotion(confirmingPromotion ? undefined : f?.id);
                      }
                    }}
                    mode={freightModeFor(id)}
                    empty={(stagesPerFreight[id] || []).length === 0}
                  >
                    <Dropdown
                      className='absolute top-2 right-2'
                      trigger={['click']}
                      menu={{
                        items: [
                          {
                            key: '1',
                            label: (
                              <>
                                <FontAwesomeIcon icon={faCircleCheck} className='mr-2' /> Manually
                                Approve
                              </>
                            ),
                            onClick: () => {
                              setManuallyApproving(id);
                            }
                          }
                        ]
                      }}
                    >
                      <FontAwesomeIcon
                        icon={faEllipsisV}
                        className='cursor-pointer text-gray-500 hover:text-white'
                      />
                    </Dropdown>
                    <StageIndicators
                      stages={stagesPerFreight[id] || []}
                      faded={!!manuallyApproving}
                    />
                    <FreightContents
                      highlighted={
                        // contains stages, not in promotion mode
                        ((stagesPerFreight[id] || []).length > 0 && !promotingStage) ||
                        // in promotion mode, is eligible
                        (!!promotingStage && promotionEligible[id]) ||
                        false
                      }
                      promoting={!!promotingStage}
                      freight={f}
                    />
                    {promotingStage && confirmingPromotion === id && (
                      <ConfirmPromotionDialogue
                        stageName={promotingStage?.metadata?.name || ''}
                        promotionType={promotionType || 'default'}
                        onClick={() => {
                          const currentData = {
                            project: promotingStage?.metadata?.namespace,
                            freight: f?.id
                          };
                          if (promotionType === 'default') {
                            promoteAction({
                              name: promotingStage?.metadata?.name,
                              ...currentData
                            });
                          } else {
                            promoteSubscribersAction({
                              stage: promotingStage?.metadata?.name,
                              ...currentData
                            });
                          }
                        }}
                      />
                    )}
                  </FreightItem>
                );
              })}
          </>
        </Freightline>
        <div className='flex flex-grow w-full'>
          <div className={`overflow-hidden flex-grow w-full ${styles.dag}`}>
            <div className='text-sm mb-4 font-semibold p-6'>
              <FontAwesomeIcon icon={faDiagramProject} className='mr-2' />
              PIPELINE
            </div>
            <div className='overflow-auto p-6'>
              <div
                className='relative'
                style={{ width: box?.width, height: box?.height, margin: '0 auto' }}
              >
                {nodes?.map((node, index) => (
                  <div
                    key={index}
                    className='absolute'
                    style={{
                      left: node.left,
                      top: node.top,
                      width: node.width,
                      height: node.height
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
                            fullFreightById[node.data?.status?.currentFreight?.id || '']
                          }
                          hasNoSubscribers={
                            (subscribersByStage[node?.data?.metadata?.name || ''] || []).length ===
                            0
                          }
                          onPromoteClick={(type: PromotionType) => {
                            if (promotingStage?.metadata?.name === node.data?.metadata?.name) {
                              setPromotingStage(undefined);
                            } else {
                              setPromotingStage(node.data);
                              setPromotionType(type);
                            }
                            setConfirmingPromotion(undefined);
                          }}
                          promoting={
                            promotingStage?.metadata?.name === node.data?.metadata?.name
                              ? promotionType
                              : undefined
                          }
                          onClick={
                            manuallyApproving
                              ? () => {
                                  manualApproveAction({
                                    stage: node.data?.metadata?.name,
                                    project: name,
                                    id: manuallyApproving
                                  });
                                }
                              : undefined
                          }
                          approving={!!manuallyApproving}
                        />
                      </>
                    ) : (
                      <RepoNode nodeData={node}>
                        <div className='flex w-full'>
                          <Button
                            onClick={() =>
                              refreshWarehouseAction({ name: node.warehouseName, project: name })
                            }
                            icon={<FontAwesomeIcon icon={faRefresh} />}
                            size='small'
                            className='mt-1 ml-auto'
                          >
                            Refresh
                          </Button>
                        </div>
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
                        height: lineThickness,
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
          <div
            className='text-gray-300 text-sm'
            style={{
              width: '400px',
              backgroundColor: '#222'
            }}
          >
            <h3 className='bg-black px-6 pb-3 pt-4 flex items-center'>
              <FontAwesomeIcon icon={faDocker} className='mr-2' /> IMAGES
            </h3>
            <div className='p-4'>
              <Images projectName={name as string} stages={sortedStages || []} />
            </div>
          </div>
        </div>
        {stage && <StageDetails stage={stage} />}
      </ColorContext.Provider>
    </div>
  );
};
