import { Edge, MarkerType, Node } from '@xyflow/react';
import { graphlib, layout } from 'dagre';
import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines-2/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines-2/context/dictionary-context';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { CUSTOM_NODE } from './constant';

export const useMiniPromotionGraph = (stage: Stage, freight: Freight) => {
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  return useMemo(() => {
    const stageName = stage?.metadata?.name || '';
    const graph = new graphlib.Graph<{ handles: number }>();

    graph.setGraph({ rankdir: 'LR' });
    graph.setDefaultEdgeLabel(() => ({}));

    graph.setNode(stageName, {
      ...nodeSize(),
      handles:
        actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM
          ? dictionaryContext?.subscribersByStage[stageName]?.size
          : 1
    });

    if (actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM) {
      const subscribers = dictionaryContext?.subscribersByStage[stageName] || new Set<string>();
      for (const stage of subscribers) {
        graph.setNode(stage, { ...nodeSize(), handles: subscribers.size });
        graph.setEdge(stageName, stage);
      }
    } else {
      const parentStages = new Set<string>();

      for (const [stage, subscribers] of Object.entries(
        dictionaryContext?.subscribersByStage || {}
      )) {
        if (subscribers.has(stageName)) {
          parentStages.add(stage);
        }
      }

      if (parentStages.size) {
        for (const parentStage of parentStages) {
          graph.setNode(parentStage, { ...nodeSize(), handles: parentStages.size });
          graph.setEdge(parentStage, stageName);
        }
      } else {
        graph.setNode(freight?.alias || '', nodeSize());
        graph.setEdge(freight?.alias, stageName);
      }
    }

    layout(graph, { lablepos: 'c' });

    const reactFlowNodes: Node[] = [];
    const reactFlowEdges: Edge[] = [];

    for (const node of graph.nodes()) {
      const dagreNode = graph.node(node);

      reactFlowNodes.push({
        id: node,
        type: CUSTOM_NODE,
        position: {
          x: dagreNode?.x,
          y: dagreNode?.y
        },
        data: {
          label: node,
          handles: dagreNode.handles
        }
      });
    }

    for (const edge of graph.edges()) {
      reactFlowEdges.push({
        id: edge?.name || '',
        source: edge.v,
        target: edge.w,
        markerEnd: {
          type: MarkerType.ArrowClosed
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [stage, freight, dictionaryContext?.subscribersByStage, actionContext?.action]);
};

const nodeSize = () => ({ width: 200, height: 100 });
