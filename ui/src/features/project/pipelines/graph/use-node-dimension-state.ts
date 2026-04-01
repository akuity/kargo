import { NodeDimensionChange } from '@xyflow/react';
import { useCallback, useState } from 'react';

export type DimensionState = Record<string, { width: number; height: number }>;

export const useNodeDimensionState = () => {
  const [dimensions, setDimensions] = useState<DimensionState>({});

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
    },
    [setDimensions]
  );

  return [dimensions, setDimensions, onNodeSizeChange] as const;
};
