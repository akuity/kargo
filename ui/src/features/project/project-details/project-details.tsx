import { createPromiseClient } from '@bufbuild/connect';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Empty } from 'antd';
import { graphlib, layout } from 'dagre';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transport } from '@ui/config/transport';
import { LoadingState } from '@ui/features/common';
import { StageDetails } from '@ui/features/stage/stage-details';
import { getStage, listStages } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { Stage } from '@ui/gen/v1alpha1/types_pb';
import { useDocumentEvent } from '@ui/utils/document';

import { StageNode } from './stage-node';

const lineThickness = 2;
const nodeWidth = 144;
const nodeHeight = 100;

// TODO: replace with real colors
const colors = [
  '#0DADEA', // blue
  '#DE7EAE', // pink
  '#FF9500', // orange
  '#4B0082', // purple
  '#F5d905', // yellow
  '#964B00' // brown
];

export const ProjectDetails = () => {
  const { name, stageName } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));
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

  const [nodes, connectors, box] = React.useMemo(() => {
    if (!data) {
      return [[], []];
    }

    const g = new graphlib.Graph();
    g.setGraph({ rankdir: 'LR' });
    g.setDefaultEdgeLabel(() => ({}));
    const colorByStage = new Map<string, string>();
    const stageByName = new Map<string, Stage>();
    const stages = data.stages
      .slice()
      .sort((a, b) => a.metadata?.name?.localeCompare(b.metadata?.name || '') || 0);

    stages?.forEach((stage, i) => {
      colorByStage.set(stage.metadata?.name || '', colors[i % colors.length]);
      stageByName.set(stage.metadata?.name || '', stage);
      g.setNode(stage.metadata?.name || '', {
        label: stage.metadata?.name || '',
        width: nodeWidth,
        height: nodeHeight
      });
    });
    stages.forEach((stage) => {
      stage?.spec?.subscriptions?.upstreamStages.forEach((item) => {
        g.setEdge(item.name || '', stage.metadata?.name || '');
      });
    });
    layout(g);

    const nodes = g.nodes().map((name) => {
      const node = g.node(name);
      return {
        left: node.x - node.width / 2,
        top: node.y - node.height / 2,
        width: node.width,
        height: node.height,
        stage: stageByName.get(name) as Stage,
        color: colorByStage.get(name) as string
      };
    });

    const connectors = g.edges().map((name) => {
      const edge = g.edge(name);
      const lines = new Array<{ x: number; y: number; width: number; angle: number }>();
      for (let i = 0; i < edge.points.length - 1; i++) {
        const start = edge.points[i];
        const end = edge.points[i + 1];
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
    return [nodes, connectors, box];
  }, [data]);

  if (isLoading) return <LoadingState />;

  if (!data || data.stages.length === 0) return <Empty />;
  const stage = stageName && data.stages.find((item) => item.metadata?.name === stageName);

  return (
    <div>
      <div
        className='relative'
        style={{ width: box?.width, height: box?.height, margin: '0 auto' }}
      >
        {nodes?.map((node) => (
          <div
            key={node.stage?.metadata?.name}
            className='absolute cursor-pointer'
            onClick={() =>
              navigate(generatePath(paths.stage, { name, stageName: node.stage.metadata?.name }))
            }
            style={{
              left: node.left,
              top: node.top,
              width: node.width,
              height: node.height
            }}
          >
            <StageNode stage={node.stage} color={node.color} />
          </div>
        ))}
        {connectors?.map((connector) =>
          connector.map((line, i) => (
            <div
              className='absolute'
              style={{
                padding: 0,
                margin: 0,
                background: 'gray',
                height: lineThickness,
                width: line.width,
                left: line.x,
                top: line.y,
                transform: `rotate(${line.angle}deg)`
              }}
              key={i}
            >
              {i === connector.length - 1 && (
                <div
                  style={{
                    position: 'absolute',
                    left: -1,
                    top: -lineThickness * 4 + 1,
                    height: 0,
                    borderTop: `${lineThickness * 4}px solid transparent`,
                    borderBottom: `${lineThickness * 4}px solid transparent`,
                    borderRight: `${lineThickness * 4}px solid gray`
                  }}
                />
              )}
            </div>
          ))
        )}
      </div>
      {stage && <StageDetails stage={stage} />}
    </div>
  );
};
