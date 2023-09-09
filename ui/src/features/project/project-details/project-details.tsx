import { createPromiseClient } from '@bufbuild/connect';
import { faDiagramProject } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Empty } from 'antd';
import { graphlib, layout } from 'dagre';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transport } from '@ui/config/transport';
import { LoadingState } from '@ui/features/common';
import { Freightline } from '@ui/features/freightline/freightline';
import { Palette } from '@ui/features/freightline/palette';
import { StageDetails } from '@ui/features/stage/stage-details';
import {
  getStage,
  listStages,
  queryFreight
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { Stage } from '@ui/gen/v1alpha1/types_pb';
import { useDocumentEvent } from '@ui/utils/document';
import { getStageColors } from '@ui/utils/stages';

import { StageNode } from './stage-node';

const lineThickness = 2;
const nodeWidth = 144;
const nodeHeight = 100;

export const ProjectDetails = () => {
  const { name, stageName } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));
  const { data: freightData, isLoading: isLoadingFreight } = useQuery(
    queryFreight.useQuery({ project: name })
  );
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
    const stageByName = new Map<string, Stage>();
    const colorByStage = new Map<string, string>();
    const stages = data.stages
      .slice()
      .sort((a, b) => a.metadata?.name?.localeCompare(b.metadata?.name || '') || 0);

    const colors = getStageColors(stages);
    stages?.forEach((stage) => {
      const curColor = colors[stage?.metadata?.uid || ''];
      colorByStage.set(stage.metadata?.name || '', curColor);
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

  const [stagesPerFreight, setStagesPerFreight] = React.useState<{ [key: string]: Stage[] }>({});
  const [stageColorMap, setStageColorMap] = React.useState<{ [key: string]: string }>({});

  React.useEffect(() => {
    const stagesPerFreight: { [key: string]: Stage[] } = {};
    setStageColorMap(getStageColors(data?.stages || []));
    (data?.stages || []).forEach((stage) => {
      const items = stagesPerFreight[stage.status?.currentFreight?.id || ''] || [];
      stagesPerFreight[stage.status?.currentFreight?.id || ''] = [...items, stage];
    });
    setStagesPerFreight(stagesPerFreight);
  }, [data, freightData]);

  if (isLoading || isLoadingFreight) return <LoadingState />;

  if (!data || data.stages.length === 0) return <Empty />;
  const stage = stageName && data.stages.find((item) => item.metadata?.name === stageName);

  return (
    <div>
      <Freightline
        freight={freightData?.groups['']?.freight || []}
        stagesPerFreight={stagesPerFreight}
        stageColorMap={stageColorMap}
      />
      <div className='mb-16 p-6'>
        <div className='text-sm mb-4'>
          <FontAwesomeIcon icon={faDiagramProject} className='mr-2' />
          STAGE GRAPH
        </div>
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
      </div>
      {stage && <StageDetails stage={stage} />}
    </div>
  );
};
