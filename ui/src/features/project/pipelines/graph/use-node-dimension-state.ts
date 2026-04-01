import { NodeDimensionChange } from '@xyflow/react';
import { useCallback, useRef, useState } from 'react';

export type DimensionState = Record<string, { width: number; height: number }>;

const transitionDurationMs = 300;

export const useNodeDimensionState = () => {
  const [dimensions, setDimensions] = useState<DimensionState>({});
  const cursorTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const onNodeSizeChange = useCallback(
    (sizeChanges: NodeDimensionChange[]) => {
      setTimeout(() => {
        setDimensions((prev) => {
          let changed = false;
          const updated = { ...prev };

          for (const change of sizeChanges) {
            if (change.dimensions) {
              // Round to integer pixels to avoid floating-point noise from
              // ResizeObserver on high-DPI (retina) displays, which would
              // otherwise trigger repeated relayouts on sub-pixel differences.
              const w = Math.round(change.dimensions.width);
              const h = Math.round(change.dimensions.height);

              if (updated[change.id]?.height !== h || updated[change.id]?.width !== w) {
                updated[change.id] = { width: w, height: h };
                changed = true;
              }
            }
          }

          return changed ? updated : prev;
        });
      });

      // When dimensions change, nodes reposition and onlyRenderVisibleElements
      // briefly unmounts nodes that shifted outside the viewport, causing the
      // pane to show a grab cursor. Override body cursor for the transition
      // duration so the user sees no flicker.
      document.body.style.cursor = 'pointer';
      if (cursorTimerRef.current) {
        clearTimeout(cursorTimerRef.current);
      }
      cursorTimerRef.current = setTimeout(() => {
        document.body.style.cursor = '';
      }, transitionDurationMs);
    },
    [setDimensions]
  );

  return [dimensions, setDimensions, onNodeSizeChange] as const;
};
