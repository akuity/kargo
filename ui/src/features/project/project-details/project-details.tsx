import { FlowAnalysisGraph, FlowGraphEdgeData, IGraph, LabelStyle } from '@ant-design/graphs';
import { useQuery } from '@tanstack/react-query';
import { Empty } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { listStages } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

export const ProjectDetails = () => {
  const { name } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));
  const graphRef = React.useRef<IGraph | undefined>();

  const nodes = React.useMemo(
    () =>
      data?.stages.flatMap((item) =>
        item.metadata?.name
          ? [
              {
                id: item.metadata?.name,
                value: {
                  title: item.metadata?.name,
                  items: [
                    {
                      text: 'Status',
                      value: item.status?.currentState?.health?.status || 'Unknown'
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
    </>
  );
};
