import { Dispatch, RefObject, SetStateAction, useCallback, useEffect, useRef } from 'react';

import { useLocalStorage } from '@ui/utils/use-local-storage';

type PipelineViewPref = {
  zoom?: number;
  // coordinates - [x, y]
  position?: [number, number];
};

export const usePipelineViewPrefHook = (project: string, opts?: { onSet?(): void }) => {
  const key = `${project}-pipeline-view-pref`;

  const [state] = useLocalStorage(key) as [
    PipelineViewPref,
    Dispatch<SetStateAction<PipelineViewPref>>
  ];

  const setState = (nextPref: PipelineViewPref) => {
    // IMPORTANT: for performance reasons we don't want react to recalculate the whole pipeline view if preference is changed
    // this is only required on first render
    window.localStorage.setItem(key, JSON.stringify(nextPref));
    opts?.onSet?.();
  };

  return [state, setState] as const;
};

const getTranslateMatrix = (node: HTMLElement) => {
  const style = window.getComputedStyle(node);

  const matrix = new DOMMatrix(style['transform']);

  return matrix;
};

type pipelineInfiniteCanvasHook = {
  refs: {
    movingObjectsRef: RefObject<HTMLDivElement>;
    zoomRef: RefObject<HTMLDivElement>;
    pipelinesConfigRef: RefObject<HTMLDivElement>;
  };
  moveSpeed?: number; // px - default 2.5
  zoomSpeed?: number; // % - default 5
  onCanvas?(node: HTMLDivElement): void;
  onMove?(newPref: PipelineViewPref): void;
  pipelineViewPref?: PipelineViewPref;
};

export const usePipelinesInfiniteCanvas = (conf: pipelineInfiniteCanvasHook) => {
  const cleanupFunction = useRef<() => void>();

  const moveSpeed = conf?.moveSpeed || 2.5;
  const zoomSpeed = conf?.zoomSpeed || 5;

  useEffect(() => {
    return cleanupFunction.current;
  }, []);

  const getCurrentZoom = useCallback(() => {
    if (!conf.refs.zoomRef.current) {
      return 100;
    }

    return getTranslateMatrix(conf.refs.zoomRef.current).a * 100;
  }, []);

  const zoom = useCallback((percentage: number) => {
    if (!conf.refs.zoomRef.current) {
      return;
    }

    conf.refs.zoomRef.current.style.transform = `scale(${percentage}%)`;
  }, []);

  const zoomOut = useCallback(() => {
    const currentZoom = getCurrentZoom();

    zoom(currentZoom + zoomSpeed);
  }, []);

  const zoomIn = useCallback(() => {
    const currentZoom = getCurrentZoom();

    zoom(currentZoom - zoomSpeed);
  }, []);

  const fitToView = useCallback((canvasNode: HTMLDivElement) => {
    if (
      !conf.refs.pipelinesConfigRef.current ||
      !conf.refs.zoomRef.current ||
      !conf.refs.movingObjectsRef.current
    ) {
      return;
    }

    // reset previously scaled properties - must
    conf.refs.zoomRef.current.style.transform = '';
    updatePos(0, 0);

    // canvas hides the overflow of pipeline so we want accurate view by screen
    const { x, y, left, top } = canvasNode.getBoundingClientRect();
    const canvasHeight = document.body.offsetHeight - y;
    const canvasWidth = document.body.offsetWidth - x;

    const pipelineConfigWidth =
      document.body.offsetWidth - conf.refs.pipelinesConfigRef.current.getBoundingClientRect().x;

    const padding = 50;

    const W2 = canvasWidth - pipelineConfigWidth - padding;
    const H2 = canvasHeight - padding;

    const pipelineRect = conf.refs.zoomRef.current.getBoundingClientRect();

    const W1 = pipelineRect.width;
    const H1 = pipelineRect.height;

    const nextZoom = Math.min(W2 / W1, H2 / H1);

    if (nextZoom === 1) {
      return;
    }

    conf.refs.zoomRef.current.style.transform = `scale(${nextZoom})`;

    // now move the pipeline to fit the screen
    const x2 = left + W2 / 2;
    const y2 = top + H2 / 2;

    const newPipelineRect /* because we did zoom */ =
      conf.refs.zoomRef.current.getBoundingClientRect();

    const x1 = newPipelineRect.left + newPipelineRect.width / 2 - padding / 2;
    const y1 = newPipelineRect.top + newPipelineRect.height / 2 - padding / 2;

    const deltaX = x2 - x1;
    const deltaY = y2 - y1;

    updatePos(deltaX, deltaY);

    conf?.onMove?.(getPipelineView());
  }, []);

  const getPos = useCallback(() => {
    if (conf.refs.movingObjectsRef.current) {
      const { e, f } = getTranslateMatrix(conf.refs.movingObjectsRef.current).translate();

      return [e, f] as const;
    }

    return [0, 0] as const;
  }, []);

  const updatePos = useCallback((x: number, y: number) => {
    const currentPos = getPos();

    const newPos = [x, y];

    if (!conf.refs.movingObjectsRef.current) {
      return;
    }

    const startTransform = `translate(${currentPos[0]}px, ${currentPos[1]}px)`;
    const endTransform = `translate(${newPos[0]}px, ${newPos[1]}px)`;

    conf.refs.movingObjectsRef.current.style.animation = 'none';
    conf.refs.movingObjectsRef.current.style.setProperty('--end-transform', endTransform);
    conf.refs.movingObjectsRef.current.style.setProperty('--start-transform', startTransform);
    conf.refs.movingObjectsRef.current.style.animation = '';
  }, []);

  const getPipelineView = useCallback(() => {
    const [x = 0, y = 0] = getPos();
    return {
      zoom: getCurrentZoom(),
      position: [x, y]
    } satisfies PipelineViewPref;
  }, []);

  const registerCanvas = useCallback((canvasNode: HTMLDivElement | null) => {
    if (!canvasNode) {
      return;
    }

    conf.onCanvas?.(canvasNode);

    const { pipelineViewPref } = conf;

    if (pipelineViewPref) {
      if (typeof pipelineViewPref?.zoom === 'number') {
        zoom(pipelineViewPref.zoom);
      }

      if (pipelineViewPref?.position?.length === 2) {
        updatePos(...pipelineViewPref.position);
      }
    } else {
      fitToView(canvasNode);
    }

    const startMovingObjects = (init: MouseEvent) => {
      let prev = init;
      return (e: MouseEvent) => {
        if (!conf.refs.movingObjectsRef.current) {
          return;
        }

        const deltaX = e.clientX - prev.clientX;
        const deltaY = e.clientY - prev.clientY;

        let [newRight, newTop] = getPos();

        if (deltaX > 0) {
          newRight += moveSpeed;
        } else if (deltaX < 0) {
          newRight -= moveSpeed;
        }

        if (deltaY > 0) {
          newTop += moveSpeed;
        } else if (deltaY < 0) {
          newTop -= moveSpeed;
        }

        updatePos(newRight, newTop);

        prev = e;
      };
    };

    let registeredEventListener = false;

    let onWindowMouseMove: (e: MouseEvent) => void = () => {};

    const onCanvasMouseDown = (e: MouseEvent) => {
      // skip if this click is on stage node
      if (
        !conf.refs.zoomRef.current?.isEqualNode(e.target as Node) &&
        !conf.refs.movingObjectsRef.current?.isEqualNode(e.target as Node)
      ) {
        return;
      }

      if (registeredEventListener) {
        onCanvasMouseUp();
        return;
      }
      registeredEventListener = true;

      if (conf.refs.zoomRef.current) {
        // block any pointer events in pipeline
        // this makes only window mousemove event happen
        // other events like hover on node will conflict and causes glitches while moving
        conf.refs.zoomRef.current.style.pointerEvents = 'none';
        conf.refs.zoomRef.current.style.cursor = 'cursor-move';
      }

      onWindowMouseMove = startMovingObjects(e);

      window.addEventListener('mousemove', onWindowMouseMove);
    };

    const onCanvasMouseUp = () => {
      registeredEventListener = false;
      conf?.onMove?.(getPipelineView());

      if (conf.refs.zoomRef.current) {
        conf.refs.zoomRef.current.style.pointerEvents = '';
        conf.refs.zoomRef.current.style.cursor = '';
      }

      window.removeEventListener('mousemove', onWindowMouseMove);
    };

    const onWheel = (e: WheelEvent) => {
      if (!conf.refs.zoomRef.current) {
        return;
      }

      if (conf.refs.pipelinesConfigRef.current) {
        const { top, height, left, width } =
          conf.refs.pipelinesConfigRef.current.getBoundingClientRect();

        const { x, y } = e;

        const overlapOnXAxis = x >= left && x <= left + width;
        const overlapOnYAxis = y >= top && y <= top + height;

        if (overlapOnXAxis && overlapOnYAxis) {
          return;
        }
      }

      if (e.deltaY > 0) {
        zoomIn();
      } else if (e.deltaY < 0) {
        zoomOut();
      }

      conf?.onMove?.(getPipelineView());
    };

    canvasNode.addEventListener('mousedown', onCanvasMouseDown);
    canvasNode.addEventListener('mouseup', onCanvasMouseUp);
    canvasNode.addEventListener('wheel', onWheel);

    cleanupFunction.current = () => {
      canvasNode.removeEventListener('mousedown', onCanvasMouseDown);
      canvasNode.removeEventListener('mouseup', onCanvasMouseUp);
      canvasNode.removeEventListener('wheel', onWheel);
    };
  }, []);

  return {
    registerCanvas,
    fitToView,
    zoomIn,
    zoomOut,
    getPipelineView
  };
};
