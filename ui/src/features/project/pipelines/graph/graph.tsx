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
import { useEffect, useMemo, useRef, useState } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { queryCache } from '@ui/features/utils/cache';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { GraphContext } from '../context/graph-context';
import { StackedNodes } from '../nodes/stacked-nodes';

import { CustomNode } from './custom-node';
import { DummyNodeRenderrer } from './dummy-node-renderrer';
import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { useEventsWatcher } from './use-events-watcher';
import { useNodeDimensionState } from './use-node-dimension-state';
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

  const [dimensions, setDimensions] = useNodeDimensionState();

  const graph = useReactFlowPipelineGraph(
    props.stages,
    props.warehouses,
    filterContext?.preferredFilter.warehouses || [],
    redraw,
    dimensions,
    {
      afterNodes: stackedNodesParents
    },
    filterContext?.preferredFilter?.hideSubscriptions
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(graph.nodes);

  useEffect(() => {
    setNodes(graph.nodes);
  }, [graph.nodes]);

  useEventsWatcher(props.project, {
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
        setRedraw(!redraw);
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
        setRedraw(!redraw);
      }

      queryCache.freight.refetch();
    }
  });

  const warehouseByName = useMemo(() => {
    const warehouseByName: Record<string, WarehouseExpanded> = {};

    for (const warehouse of props.warehouses) {
      warehouseByName[warehouse.metadata?.name || ''] = warehouse;
    }

    return warehouseByName;
  }, [props.warehouses]);

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

  return (
    <GraphContext.Provider value={{ warehouseByName, stackedNodesParents, onStack, onUnstack }}>
      <ReactFlow
        nodes={nodes}
        edges={graph.edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{
          nodes: nodesExcludingSubscriptionNodes
        }}
        proOptions={{ hideAttribution: true }}
        minZoom={0}
        onNodesChange={onNodesChange}
        onlyRenderVisibleElements
        onInit={(inst) => (reactFlowInstance.current = inst)}
      >
        {!Object.keys(dimensions).length && (
          <div className='opacity-0 overflow-hidden h-0'>
            <DummyNodeRenderrer
              stages={props.stages}
              warehouses={props.warehouses}
              onDimensionChange={setDimensions}
            />
          </div>
        )}
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
