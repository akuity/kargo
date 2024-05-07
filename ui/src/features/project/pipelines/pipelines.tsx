import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import {
  faChevronDown,
  faCircleCheck,
  faClipboard,
  faCopy,
  faEllipsisV,
  faEye,
  faEyeSlash,
  faMasksTheater,
  faPalette,
  faPencil,
  faRefresh,
  faWandSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Space, Tooltip, message } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
import { getAlias } from '@ui/features/common/utils';
import { FreightDetails } from '@ui/features/freight/freight-details';
import { ConfirmPromotionDialogue } from '@ui/features/freightline/confirm-promotion-dialogue';
import { FreightContents } from '@ui/features/freightline/freight-contents';
import { FreightItem } from '@ui/features/freightline/freight-item';
import { Freightline } from '@ui/features/freightline/freightline';
import { FreightlineHeader } from '@ui/features/freightline/freightline-header';
import { StageIndicators } from '@ui/features/freightline/stage-indicators';
import { StageDetails } from '@ui/features/stage/stage-details';
import { clearColors } from '@ui/features/stage/utils';
import { Time } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import {
  approveFreight,
  listStages,
  listWarehouses,
  promoteToStage,
  promoteToStageSubscribers,
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
import { FreightMode, FreightlineAction, NodeType } from './types';
import { UpdateFreightAliasModal } from './update-freight-alias-modal';
import { usePipelineGraph } from './use-pipeline-graph';
import { LINE_THICKNESS, WAREHOUSE_NODE_HEIGHT } from './utils/graph';
import { Watcher } from './utils/watcher';

const getSeconds = (ts?: Time): number => Number(ts?.seconds) || 0;

export const Pipelines = () => {
  const { name, stageName, freightName } = useParams();
  const { data, isLoading } = useQuery(listStages, { project: name });
  const {
    data: freightData,
    isLoading: isLoadingFreight,
    refetch: refetchFreightData
  } = useQuery(queryFreight, { project: name });

  const { data: warehouseData } = useQuery(listWarehouses, {
    project: name
  });

  const navigate = useNavigate();

  const { show: showCreateStage } = useModal(
    name ? (p) => <CreateStageModal {...p} project={name} /> : undefined
  );
  const { show: showCreateWarehouse } = useModal(
    name ? (p) => <CreateWarehouseModal {...p} project={name} /> : undefined
  );

  const isVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  const { mutate: refreshWarehouseAction } = useMutation(refreshWarehouse, {
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
      setPromotingStage(undefined);
      refetchFreightData();
    }
  });

  const [highlightedStages, setHighlightedStages] = React.useState<{ [key: string]: boolean }>({});
  const [hideSubscriptions, setHideSubscriptions] = useLocalStorage(
    `${name}-hideSubscriptions`,
    false
  );

  const { show } = useModal();
  const client = useQueryClient();

  React.useEffect(() => {
    if (!data || !isVisible || !warehouseData || !name) {
      return;
    }

    const watcher = new Watcher(name, client);
    watcher.watchStages(data.stages.slice());
    watcher.watchWarehouses(warehouseData?.warehouses || [], refetchAvailableFreight);

    return () => watcher.cancelWatch();
  }, [isLoading, isVisible, name]);

  const [nodes, connectors, box, sortedStages, stageColorMap] = usePipelineGraph(
    name,
    data?.stages || [],
    warehouseData?.warehouses || [],
    hideSubscriptions
  );

  const [stagesPerFreight, setStagesPerFreight] = React.useState<{ [key: string]: Stage[] }>({});
  const [promotingStage, setPromotingStage] = React.useState<Stage | undefined>();
  const [freightAction, setFreightAction] = React.useState<FreightlineAction | undefined>();
  const [confirmingPromotion, setConfirmingPromotion] = React.useState<string | undefined>();
  const [subscribersByStage, setSubscribersByStage] = React.useState<{ [key: string]: Stage[] }>(
    {}
  );
  const [fullFreightById, setFullFreightById] = React.useState<{ [key: string]: Freight }>({});
  const [promotionEligible, setPromotionEligible] = React.useState<{ [key: string]: boolean }>({});
  const [manuallyApproving, setManuallyApproving] = React.useState<string | undefined>();

  const { mutate: manualApproveAction } = useMutation(approveFreight, {
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(`Freight ${manuallyApproving} has been manually approved.`);
      setManuallyApproving(undefined);
      setFreightAction(undefined);
    }
  });

  const { mutate: promoteToStageSubscribersAction } = useMutation(promoteToStageSubscribers, {
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `Promotion requests to all subscribers of "${promotingStage?.metadata?.name}" have been submitted.`
      );
      setPromotingStage(undefined);
    }
  });

  const { mutate: promoteAction } = useMutation(promoteToStage, {
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `Promotion request for stage "${promotingStage?.metadata?.name}" has been successfully submitted.`
      );
      setPromotingStage(undefined);
    }
  });

  const {
    data: availableFreightData,
    refetch: refetchAvailableFreight,
    isLoading: isLoadingAvailableFreight
  } = useQuery(queryFreight, { project: name, stage: promotingStage?.metadata?.name || '' });

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
        freightAction === 'promote'
          ? initFreight
          : // if promoting subscribers, only include freight that has been verified in the promoting stage
            initFreight.filter(
              (f) => !!f?.status?.verifiedIn[promotingStage?.metadata?.name || '']
            );

      const pe: { [key: string]: boolean } = {};
      ((availableFreight as Freight[]) || []).forEach((f: Freight) => {
        pe[f?.metadata?.name || ''] = true;
      });
      setPromotionEligible(pe);
    }
  }, [availableFreightData]);

  React.useEffect(() => {
    refetchAvailableFreight();
  }, [promotingStage, freightAction, freightData]);

  React.useEffect(() => {
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
    setStagesPerFreight(stagesPerFreight);
    setSubscribersByStage(subscribersByStage);
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

  const isFaded = (stage: Stage): boolean => {
    if (!promotingStage || !confirmingPromotion) {
      return false;
    }
    if (freightAction === 'promote') {
      return promotingStage?.metadata?.name !== stage?.metadata?.name;
    }
    if (freightAction === 'promoteSubscribers') {
      return !subscribersByStage[promotingStage?.metadata?.name || '']?.find(
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
        <FreightlineHeader
          promotingStage={promotingStage}
          action={freightAction}
          cancel={() => {
            setPromotingStage(undefined);
            setFreightAction(undefined);
            setManuallyApproving(undefined);
            setConfirmingPromotion(undefined);
          }}
          downstreamSubs={(subscribersByStage[promotingStage?.metadata?.name || ''] || []).map(
            (s) => s.metadata?.name || ''
          )}
        />
        <Freightline promotingStage={promotingStage} setPromotingStage={setPromotingStage}>
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
                        setConfirmingPromotion(confirmingPromotion ? undefined : f?.metadata?.name);
                      } else {
                        navigate(generatePath(paths.freight, { name, freightName: id }));
                      }
                    }}
                    mode={freightModeFor(id)}
                    empty={(stagesPerFreight[id] || []).length === 0}
                    onHover={(h) => onHover(h, id)}
                    highlighted={(stagesPerFreight[id] || []).reduce((h, cur) => {
                      if (h) {
                        return true;
                      }
                      return highlightedStages[cur.metadata?.name || ''];
                    }, false)}
                  >
                    <Dropdown
                      className='absolute top-2 right-2 pl-2'
                      trigger={['click']}
                      dropdownRender={(menu) => {
                        return <div onClick={(e) => e.stopPropagation()}>{menu}</div>;
                      }}
                      menu={{
                        items: [
                          {
                            key: '1',
                            label: (
                              <>
                                <FontAwesomeIcon icon={faCircleCheck} className='mr-2' />
                                Manually Approve
                              </>
                            ),
                            onClick: () => {
                              setFreightAction('manualApproval');
                              setManuallyApproving(id);
                            }
                          },
                          {
                            key: '2',
                            label: (
                              <>
                                <FontAwesomeIcon icon={faClipboard} className='mr-2' /> Copy ID
                              </>
                            ),
                            onClick: () => {
                              navigator.clipboard.writeText(id);
                              message.success('Copied Freight ID to clipboard');
                            }
                          },
                          getAlias(f)
                            ? {
                                key: '3',
                                label: (
                                  <>
                                    <FontAwesomeIcon icon={faCopy} className='mr-2' /> Copy Alias
                                  </>
                                ),
                                onClick: () => {
                                  navigator.clipboard.writeText(getAlias(f) || '');
                                  message.success('Copied Freight Alias to clipboard');
                                }
                              }
                            : null,
                          {
                            key: '4',
                            label: (
                              <>
                                <FontAwesomeIcon icon={faPencil} className='mr-2' /> Change Alias
                              </>
                            ),
                            onClick: async () => {
                              show((p) => (
                                <UpdateFreightAliasModal
                                  {...p}
                                  freight={f || undefined}
                                  project={name || ''}
                                  onSubmit={() => {
                                    refetchFreightData();
                                    p.hide();
                                  }}
                                />
                              ));
                            }
                          }
                        ]
                      }}
                    >
                      <FontAwesomeIcon
                        onClick={(e) => e.stopPropagation()}
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
                      freight={f}
                    />
                    {promotingStage && confirmingPromotion === id && (
                      <ConfirmPromotionDialogue
                        stageName={promotingStage?.metadata?.name || ''}
                        promotionType={freightAction || 'default'}
                        onClick={() => {
                          const currentData = {
                            project: promotingStage?.metadata?.namespace,
                            freight: f?.metadata?.name
                          };
                          if (freightAction === 'promote') {
                            promoteAction({
                              stage: promotingStage?.metadata?.name,
                              ...currentData
                            });
                          } else {
                            promoteToStageSubscribersAction({
                              stage: promotingStage?.metadata?.name,
                              ...currentData
                            });
                          }
                          setFreightAction(undefined);
                        }}
                      />
                    )}
                  </FreightItem>
                );
              })}
          </>
        </Freightline>
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
                            fullFreightById[node.data?.status?.currentFreight?.name || '']
                          }
                          hasNoSubscribers={
                            (subscribersByStage[node?.data?.metadata?.name || ''] || []).length <= 1
                          }
                          onPromoteClick={(type: FreightlineAction) => {
                            if (promotingStage?.metadata?.name === node.data?.metadata?.name) {
                              setPromotingStage(undefined);
                              setFreightAction(undefined);
                            } else {
                              setPromotingStage(node.data);
                              setFreightAction(type);
                              if (type === 'promoteSubscribers') {
                                setConfirmingPromotion(node.data?.status?.currentFreight?.name);
                              }
                            }
                            setConfirmingPromotion(undefined);
                          }}
                          action={
                            promotingStage?.metadata?.name === node.data?.metadata?.name
                              ? freightAction
                              : undefined
                          }
                          onClick={
                            manuallyApproving
                              ? () => {
                                  manualApproveAction({
                                    stage: node.data?.metadata?.name,
                                    project: name,
                                    name: manuallyApproving
                                  });
                                }
                              : undefined
                          }
                          onHover={(h) => onHover(h, node.data?.metadata?.name || '', true)}
                          approving={!!manuallyApproving}
                          highlighted={highlightedStages[node.data?.metadata?.name || '']}
                        />
                      </>
                    ) : (
                      <RepoNode nodeData={node}>
                        {node.type === NodeType.WAREHOUSE && (
                          <div className='flex w-full h-full'>
                            <Button
                              onClick={() =>
                                refreshWarehouseAction({
                                  name: node.warehouseName,
                                  project: name
                                })
                              }
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
          <div
            className='text-neutral-600 text-sm bg-neutral-100'
            style={{
              width: '400px'
            }}
          >
            <h3 className='bg-neutral-200 px-4 py-2 flex items-center text-sm text-neutral-500'>
              <FontAwesomeIcon icon={faDocker} className='mr-2' /> IMAGES
            </h3>
            <div className='p-4'>
              <Images projectName={name as string} stages={sortedStages || []} />
            </div>
          </div>
        </div>
        {stage && <StageDetails stage={stage} />}
        {freight && <FreightDetails freight={freight} />}
      </ColorContext.Provider>
    </div>
  );
};
