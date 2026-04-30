import { graphlib, layout } from '@dagrejs/dagre';
import { Edge, MarkerType, Node } from '@xyflow/react';
import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { CUSTOM_NODE } from './constant';

export const useMiniPromotionGraph = (stage: Stage, freight: Freight) => {
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  return useMemo(() => {
    const stageName = stage?.metadata?.name || '';
    const graph = new graphlib.Graph<{ handles: number; namespace: string }>();

    graph.setGraph({ rankdir: 'LR' });
    graph.setDefaultEdgeLabel(() => ({}));

    graph.setNode(stageName, {
      ...nodeSize(),
      handles:
        actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM
          ? dictionaryContext?.subscribersByStage[stageName]?.size
          : 1,
      namespace: stage?.metadata?.namespace
    });

    let edgeHandleIdBuckets: Record<'source' | 'target', Record<string, number>> = {
      source: {},
      target: {}
    };

    const addToBucket = (type: 'source' | 'target', node: string) => {
      if (edgeHandleIdBuckets[type][node] === undefined) {
        edgeHandleIdBuckets[type] = {
          ...edgeHandleIdBuckets[type],
          [node]: -1
        };
      }

      edgeHandleIdBuckets = {
        ...edgeHandleIdBuckets,
        [type]: {
          ...edgeHandleIdBuckets[type],
          [node]: edgeHandleIdBuckets[type][node] + 1
        }
      };
    };

    const useBucket = (type: 'source' | 'target', node: string): string => {
      const id = edgeHandleIdBuckets[type][node];

      edgeHandleIdBuckets = {
        ...edgeHandleIdBuckets,
        [type]: {
          ...edgeHandleIdBuckets[type],
          [node]: edgeHandleIdBuckets[type][node] - 1
        }
      };

      return `${id < 0 ? 0 : id}`;
    };

    if (actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM) {
      const subscribers = dictionaryContext?.subscribersByStage[stageName] || new Set<string>();
      for (const stage of subscribers) {
        graph.setNode(stage, {
          ...nodeSize(),
          handles: subscribers.size,
          namespace: dictionaryContext?.stageByName?.[stage]?.metadata?.namespace
        });
        graph.setEdge(stageName, stage);
        addToBucket('source', stageName);
        addToBucket('target', stage);
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
          graph.setNode(parentStage, {
            ...nodeSize(),
            handles: parentStages.size,
            namespace: dictionaryContext?.stageByName?.[parentStage]?.metadata?.namespace
          });
          graph.setEdge(parentStage, stageName);
          addToBucket('source', parentStage);
          addToBucket('target', stageName);
        }
      } else {
        graph.setNode(freight?.alias || '', nodeSize());
        graph.setEdge(freight?.alias, stageName);
        addToBucket('source', freight?.alias);
        addToBucket('target', stageName);
      }
    }

    layout(graph, { disableOptimalOrderHeuristic: true });

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
          handles: dagreNode.handles,
          namespace: dagreNode.namespace
        }
      });
    }

    for (const edge of graph.edges()) {
      const sourceHandle = useBucket('source', edge.v);
      const targetHandle = useBucket('target', edge.w);

      reactFlowEdges.push({
        id: `${sourceHandle}->${targetHandle}`,
        type: 'smoothstep',
        source: edge.v,
        target: edge.w,
        sourceHandle,
        targetHandle,
        markerEnd: {
          type: MarkerType.ArrowClosed
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [
    stage,
    freight,
    dictionaryContext?.subscribersByStage,
    actionContext?.action,
    dictionaryContext?.stageByName
  ]);
};

const nodeSize = () => ({ width: 200, height: 100 });
