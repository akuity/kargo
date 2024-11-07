import { RefObject, useCallback, useEffect, useRef } from 'react';

// we cannot take canvas as ref as the ref might not be available on first render
// this is safe because once parent node (here canvasNode) is available, we are sure that child node is available as well
export const usePipelinesInfiniteCanvas = (conf: {
  refs: {
    movingObjectsRef: RefObject<HTMLDivElement>;
    zoomRef: RefObject<HTMLDivElement>;
    pipelinesConfigRef: RefObject<HTMLDivElement>;
  };
  moveSpeed?: number; // px - default 3
  zoomSpeed?: number; // % - default 2.5
  onCanvas?(node: HTMLDivElement): void;
}) => {
  const cleanupFunction = useRef<() => void>();

  const moveSpeed = conf?.moveSpeed || 3;
  const zoomSpeed = conf?.zoomSpeed || 2.5;

  useEffect(() => {
    return cleanupFunction.current;
  }, []);

  const zoomOut = useCallback(() => {
    if (!conf.refs.zoomRef.current) {
      return;
    }

    let currentZoom =
      (
        conf.refs.zoomRef.current.computedStyleMap().get('transform') as CSSTransformValue
      ).toMatrix().a * 100;

    currentZoom += zoomSpeed;

    conf.refs.zoomRef.current.style.transform = `scale(${currentZoom}%)`;
  }, []);

  const zoomIn = useCallback(() => {
    if (!conf.refs.zoomRef.current) {
      return;
    }

    let currentZoom =
      (
        conf.refs.zoomRef.current.computedStyleMap().get('transform') as CSSTransformValue
      ).toMatrix().a * 100;

    currentZoom -= zoomSpeed;

    conf.refs.zoomRef.current.style.transform = `scale(${currentZoom}%)`;
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
  }, []);

  const getPos = useCallback(() => {
    if (conf.refs.movingObjectsRef.current) {
      const transform = conf.refs.movingObjectsRef.current
        .computedStyleMap()
        .get('transform') as CSSTransformValue;

      if (!(transform instanceof CSSTransformValue)) {
        throw new Error(
          'Canvas moving mechanism seems to be changed and unsupported! Please report this bug.'
        );
      }

      const { e, f } = transform.toMatrix().translate();

      return [e, f];
    }

    return [0, 0];
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

  const registerCanvas = useCallback((canvasNode: HTMLDivElement | null) => {
    if (!canvasNode) {
      return;
    }

    conf.onCanvas?.(canvasNode);

    fitToView(canvasNode);

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
      if (registeredEventListener) {
        onCanvasMouseUp();
        return;
      }
      registeredEventListener = true;
      onWindowMouseMove = startMovingObjects(e);
      window.addEventListener('mousemove', onWindowMouseMove);
    };

    const onCanvasMouseUp = () => {
      registeredEventListener = false;
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
    zoomOut
  };
};
