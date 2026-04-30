import { faMap } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  ControlButton,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowInstance,
  useNodesState
} from '@xyflow/react';
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { queryCache } from '@ui/features/utils/cache';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { GraphContext } from '../context/graph-context';
import { StackedNodes } from '../nodes/stacked-nodes';

import { CustomNode } from './custom-node';
import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { useEventsWatcher } from './use-events-watcher';
import { reactFlowNodeConstants, useReactFlowPipelineGraph } from './use-pipeline-graph';

type GraphProps = {
  warehouses: WarehouseExpanded[];
  stages: Stage[];
  project: string;
};

const nodeTypes = {
  [reactFlowNodeConstants.CUSTOM_NODE]: CustomNode,
  [reactFlowNodeConstants.STACKED_NODE]: StackedNodes
};

export const Graph = (props: GraphProps) => {
  const reactFlowInstance = useRef<ReactFlowInstance | null>(null);
  const filterContext = useFreightTimelineControllerContext();

  const stackedNodesParents = filterContext?.preferredFilter?.stackedNodesParents || [];

  const setStackedNodesParents = (nodes: string[]) =>
    filterContext?.setPreferredFilter({
      ...filterContext?.preferredFilter,
      stackedNodesParents: nodes
    });

  const onStack = (parentNode: string) => {
    if (!stackedNodesParents.includes(parentNode)) {
      setStackedNodesParents([...stackedNodesParents, parentNode]);
    }
  };

  const onUnstack = (parentNode: string) => {
    setStackedNodesParents(stackedNodesParents.filter((node) => node !== parentNode));
  };

  const [redraw, setRedraw] = useState(false);

  // Cheap placeholders on first paint; swap to real components on the next tick.
  // ReactFlow mounts every node at least once for measurement, so this avoids
  // paying the cost of rendering full node bodies for items that may end up
  // off-screen and culled by onlyRenderVisibleElements.
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setReady(true), 10);
    return () => clearTimeout(timer);
  }, []);

  const graph = useReactFlowPipelineGraph(
    props.stages,
    props.warehouses,
    filterContext?.preferredFilter.warehouses || [],
    redraw,
    {
      afterNodes: stackedNodesParents
    },
    filterContext?.preferredFilter?.hideSubscriptions
  );

  const [nodes, setNodes] = useNodesState(graph.nodes);

  // useLayoutEffect fires synchronously after DOM mutations but before the browser
  // paints. This prevents the one-frame intermediate state where node content has
  // changed (and grown) but positions haven't been recalculated yet.
  useLayoutEffect(() => {
    setNodes(graph.nodes);
  }, [graph.nodes]);

  useEventsWatcher(
    props.project,
    {
      onStage(stage) {
        const index = stageIndexer.index(stage);
        setNodes((nodes) =>
          nodes.map((node) => {
            if (node.id === index && node.type === reactFlowNodeConstants.CUSTOM_NODE) {
              return {
                ...node,
                data: {
                  ...node.data,
                  value: stage
                }
              };
            }

            return node;
          })
        );

        if (!nodes.find((n) => n.id === index)) {
          setRedraw((prev) => !prev);
        }

        queryCache.imageStageMatrix.update(stage);
      },
      onWarehouse(warehouse) {
        const index = warehouseIndexer.index(warehouse);
        setNodes((nodes) =>
          nodes.map((node) => {
            if (node.id === index && node.type === reactFlowNodeConstants.CUSTOM_NODE) {
              return {
                ...node,
                data: {
                  ...node.data,
                  value: warehouse
                }
              };
            }

            return node;
          })
        );

        if (!nodes.find((n) => n.id === index)) {
          setRedraw((prev) => !prev);
        }

        queryCache.freight.refetch();
      }
    },
    filterContext?.preferredFilter?.warehouses || []
  );

  const warehouseByName = useMemo(() => {
    const warehouseByName: Record<string, WarehouseExpanded> = {};

    for (const warehouse of props.warehouses) {
      warehouseByName[warehouse.metadata?.name || ''] = warehouse;
    }

    return warehouseByName;
  }, [props.warehouses]);

  const [hoveredWarehouseName, setHoveredWarehouseName] = useState<string | null>(null);

  const distinctEdgeWarehouses = useMemo(
    () =>
      new Set(graph.edges.map((e) => e.data?.warehouseName as string | undefined).filter(Boolean))
        .size,
    [graph.edges]
  );

  const edges = useMemo(() => {
    if (!hoveredWarehouseName || distinctEdgeWarehouses < 2) {
      return graph.edges;
    }
    return graph.edges.map((edge) => {
      if (edge.data?.warehouseName !== hoveredWarehouseName) {
        return edge;
      }
      const color = (edge.style?.stroke as string) || '#000';
      return {
        ...edge,
        style: {
          ...edge.style,
          strokeOpacity: 0.8,
          filter: `drop-shadow(0 0 7px ${color}50)`
        }
      };
    });
  }, [graph.edges, hoveredWarehouseName, distinctEdgeWarehouses]);

  const nodesExcludingSubscriptionNodes = useMemo(() => {
    const subscriptionNodes = nodes.filter((n) => repoSubscriptionIndexer.is(n.id));

    if (subscriptionNodes?.length > 5) {
      return nodes.filter((n) => !repoSubscriptionIndexer.is(n.id));
    }

    return nodes;
  }, [nodes]);

  useEffect(() => {
    requestAnimationFrame(() => {
      reactFlowInstance.current?.fitView();
    });
  }, [filterContext?.preferredFilter?.hideSubscriptions]);

  const loggedRef = useRef(false);
  const mountTimeRef = useRef(performance.now());
  useEffect(() => {
    if (loggedRef.current || graph.nodes.length === 0) {
      return;
    }
    loggedRef.current = true;
    requestAnimationFrame(() => {
      // eslint-disable-next-line no-console
      console.log(
        `[Graph] painted in ${((performance.now() - mountTimeRef.current) / 1000).toFixed(3)}s`
      );
    });
  }, [graph.nodes]);

  return (
    <GraphContext.Provider
      value={{
        warehouseByName,
        stackedNodesParents,
        onStack,
        onUnstack,
        ready,
        hoveredWarehouseName,
        setHoveredWarehouseName
      }}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{
          nodes: nodesExcludingSubscriptionNodes
        }}
        proOptions={{ hideAttribution: true }}
        minZoom={nodes.length > 100 ? 0.4 : 0.1}
        maxZoom={1.4}
        onlyRenderVisibleElements
        panOnDrag
        onInit={(inst) => (reactFlowInstance.current = inst)}
      >
        {filterContext?.preferredFilter?.showMinimap && (
          <MiniMap
            style={{ background: 'white', border: '1px solid lightblue', borderRadius: '5px' }}
            pannable
          />
        )}
        <Controls
          showInteractive={false}
          onFitView={() => {
            setRedraw(!redraw);
          }}
        >
          <ControlButton
            title={filterContext?.preferredFilter?.showMinimap ? 'Hide Minimap' : 'Show Minimap'}
            onClick={() =>
              filterContext?.setPreferredFilter({
                ...filterContext?.preferredFilter,
                showMinimap: !filterContext?.preferredFilter?.showMinimap
              })
            }
          >
            <FontAwesomeIcon icon={faMap} />
          </ControlButton>
        </Controls>
      </ReactFlow>
    </GraphContext.Provider>
  );
};
