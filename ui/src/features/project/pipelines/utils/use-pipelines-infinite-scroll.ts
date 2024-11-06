import { RefObject, useCallback, useEffect, useRef } from 'react';

// we cannot take canvas as ref as the ref might not be available on first render
// this is safe because once parent node (here canvasNode) is available, we are sure that child node is available as well
export const usePipelinesInfiniteScroll = (conf: {
  refs: {
    movingObjectsRef: RefObject<HTMLDivElement>;
    zoomRef: RefObject<HTMLDivElement>;
  };
  moveSpeed?: number; // px - default 3
  zoomSpeed?: number; // % - default 2.5
}) => {
  const cleanupFunction = useRef<() => void>();

  useEffect(() => {
    return cleanupFunction.current;
  }, []);

  return useCallback((canvasNode: HTMLDivElement | null) => {
    if (!canvasNode) {
      return;
    }
    const moveSpeed = conf?.moveSpeed || 3;
    const zoomSpeed = conf?.zoomSpeed || 10;

    const startMovingObjects = (init: MouseEvent) => {
      let prev = init;
      return (e: MouseEvent) => {
        if (!conf.refs.movingObjectsRef.current) {
          return;
        }

        const deltaX = e.clientX - prev.clientX;
        const deltaY = e.clientY - prev.clientY;

        let newTop = Number(
          conf.refs.movingObjectsRef.current?.style.top.slice(
            0,
            -2
          ) /* slice(0, -2) = without "px" */
        );
        let newRight = Number(conf.refs.movingObjectsRef?.current?.style.right.slice(0, -2));

        if (deltaX > 0) {
          newRight -= moveSpeed;
        } else if (deltaX < 0) {
          newRight += moveSpeed;
        }

        if (deltaY > 0) {
          newTop += moveSpeed;
        } else if (deltaY < 0) {
          newTop -= moveSpeed;
        }

        conf.refs.movingObjectsRef.current.style.right = `${newRight}px`;
        conf.refs.movingObjectsRef.current.style.top = `${newTop}px`;

        prev = e;
      };
    };

    let onWindowMouseMove: (e: MouseEvent) => void = () => {};

    const onCanvasMouseDown = (e: MouseEvent) => {
      onWindowMouseMove = startMovingObjects(e);
      window.addEventListener('mousemove', onWindowMouseMove);
    };

    const onCanvasMouseUp = () => {
      window.removeEventListener('mousemove', onWindowMouseMove);
    };

    const onWheel = (e: WheelEvent) => {
      if (!conf.refs.zoomRef.current) {
        return;
      }
      let currentZoom = Number(conf.refs.zoomRef.current.style.zoom.slice(0, -1)) || 100;

      if (e.deltaY > 0) {
        currentZoom -= zoomSpeed;
      } else if (e.deltaY < 0) {
        currentZoom += zoomSpeed;
      }

      conf.refs.zoomRef.current.style.zoom = `${currentZoom}%`;
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
};
