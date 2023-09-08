import {
  CardItems,
  CustomCfg,
  FlowAnalysisGraph,
  FlowGraphEdgeData,
  IGraph,
  IGroup,
  LabelStyle
} from '@ant-design/graphs';
import { createPromiseClient } from '@bufbuild/connect';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Empty } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transport } from '@ui/config/transport';
import { LoadingState } from '@ui/features/common';
import { healthStateToString } from '@ui/features/common/health-status/utils';
import { StageDetails } from '@ui/features/stage/stage-details';
import { getStage, listStages } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { Stage } from '@ui/gen/v1alpha1/types_pb';
import { useDocumentEvent } from '@ui/utils/document';

import { healthStateToIcon } from './utils/health';

export const ProjectDetails = () => {
  const { name, stageName } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));
  const graphRef = React.useRef<IGraph | undefined>();
  const client = useQueryClient();

  const isVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  React.useEffect(() => {
    if (!data || !isVisible) {
      return;
    }

    const cancel = new AbortController();

    const watchStages = async () => {
      const promiseClient = createPromiseClient(KargoService, transport);
      const stream = promiseClient.watchStages({ project: name }, { signal: cancel.signal });

      for await (const e of stream) {
        const index = data.stages.findIndex(
          (item) => item.metadata?.name === e.stage?.metadata?.name
        );
        let stages = data.stages;
        if (e.type === 'DELETED') {
          if (index !== -1) {
            stages = [...stages.slice(0, index), ...data.stages.slice(index + 1)];
          }
        } else {
          if (index === -1) {
            stages = [...stages, e.stage as Stage];
          } else {
            stages = [
              ...data.stages.slice(0, index),
              e.stage as Stage,
              ...data.stages.slice(index + 1)
            ];
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

  const nodes = React.useMemo(
    () =>
      data?.stages
        .sort((a, b) => a.metadata?.name?.localeCompare(b.metadata?.name || '') || 0)
        .flatMap((item) =>
          item.metadata?.name
            ? [
                {
                  id: item.metadata?.name,
                  value: {
                    title: item.metadata?.name,
                    items: [
                      {
                        text: 'Status',
                        value: healthStateToString(item.status?.currentFreight?.health?.status),
                        icon: healthStateToIcon(item.status?.currentFreight?.health?.status)
                      }
                    ]
                  }
                }
              ]
            : []
        ) || [],
    [data]
  );

  const edges = React.useMemo(
    () =>
      data?.stages.reduce<FlowGraphEdgeData[]>((acc, curr) => {
        if (curr.spec?.subscriptions?.upstreamStages.length) {
          return [
            ...acc,
            ...curr.spec.subscriptions.upstreamStages.flatMap((item) =>
              item.name && curr.metadata?.name
                ? [
                    {
                      source: item.name,
                      target: curr.metadata?.name
                    }
                  ]
                : []
            )
          ];
        }

        return acc;
      }, []) || [],
    [data]
  );

  React.useEffect(() => {
    // Hacky way to recenter graph after adding a new stage
    setTimeout(() => graphRef.current?.fitCenter?.(), 0);
  }, [nodes, edges]);

  if (isLoading) return <LoadingState />;

  if (!data || data.stages.length === 0) return <Empty />;
  const stage = stageName && data.stages.find((item) => item.metadata?.name === stageName);

  return (
    <>
      <FlowAnalysisGraph
        behaviors={['drag-canvas', 'zoom-canvas']}
        data={{
          nodes,
          edges
        }}
        autoFit={false}
        animate={false}
        height={600}
        edgeCfg={{
          edgeStateStyles: { hover: { stroke: '#ccc', lineWidth: 1 } },
          endArrow: { show: true },
          type: 'polyline'
        }}
        markerCfg={{ show: false }}
        layout={
          {
            ranksepFunc: () => 40,
            nodesepFunc: () => 10
          } as any // eslint-disable-line @typescript-eslint/no-explicit-any
        }
        nodeCfg={{
          size: [180, 40],
          hover: {
            fill: '#ccc',
            lineWidth: 1
          },
          style: { cursor: 'pointer', stroke: '#e8e8e8', radius: 4 },
          padding: 12,
          label: { style: { cursor: 'pointer', fontSize: 14 } as LabelStyle },
          customContent: (item: CardItems, group: IGroup, cfg: CustomCfg) => {
            const { startX, startY } = cfg;

            if (!item.icon || !item.value) {
              return;
            }

            group.addShape('image', {
              attrs: {
                x: startX,
                y: startY,
                img: item.icon,
                width: 20,
                height: 20,
                cursor: 'pointer'
              },
              name: `image-${Math.random()}`
            });

            group?.addShape('text', {
              attrs: {
                textBaseline: 'top',
                x: (startX || 0) + 28,
                y: (startY || 0) + 4,
                text: item.value,
                fill: '#5a5a5a',
                cursor: 'pointer',
                fontSize: 13
              },
              name: `text-${Math.random()}`
            });

            return 26;
          },
          title: {
            style: {
              cursor: 'pointer',
              fontSize: 16,
              y: 8
            },
            containerStyle: {
              radius: 4,
              cursor: 'pointer',
              fill: '#254166',
              y: -4
            }
          },
          nodeStateStyles: {
            hover: {
              stroke: '#e8e8e8',
              fill: '#f5f5f5',
              lineWidth: 1
            }
          }
        }}
        toolbarCfg={{ show: true }}
        onReady={(graph) => {
          graphRef.current = graph;
          graph.on('node:click', (evt) => {
            evt.item?._cfg?.id &&
              navigate(generatePath(paths.stage, { name, stageName: evt.item?._cfg?.id }));
          });
        }}
      />
      {stage && <StageDetails stage={stage} />}
    </>
  );
};
