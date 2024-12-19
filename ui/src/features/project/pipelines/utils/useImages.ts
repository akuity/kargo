import { useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { getCurrentFreight } from '@ui/features/common/utils';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

import { StageStyleMap } from '../types';

export const useImages = (stages: Stage[]) => {
  const { stageColorMap } = useContext(ColorContext);

  return useMemo(() => {
    const images = new Map<string, Map<string, StageStyleMap>>();
    stages.forEach((stage) => {
      const len = stage.status?.freightHistory?.length || 0;
      stage.status?.freightHistory?.forEach((freight, i) => {
        for (const warehouseValue of Object.values(freight?.items)) {
          warehouseValue.images?.forEach((image) => {
            let repo = image.repoURL ? images.get(image.repoURL) : undefined;
            if (!repo) {
              repo = new Map<string, StageStyleMap>();
              images.set(image.repoURL!, repo);
            }
            let curStages = image.tag ? repo.get(image.tag) : undefined;
            if (!curStages) {
              curStages = {} as StageStyleMap;
            }
            curStages[stage.metadata?.name as string] = {
              opacity: 1 - i / len,
              backgroundColor: stageColorMap[stage.metadata?.name as string]
            };
            repo.set(image.tag!, curStages);
          });
        }
      });

      const existingImages = getCurrentFreight(stage).flatMap((freight) => freight.images || []);
      (existingImages || []).forEach((image) => {
        let repo = image.repoURL ? images.get(image.repoURL) : undefined;
        if (!repo) {
          repo = new Map<string, StageStyleMap>();
          images.set(image.repoURL!, repo);
        }
        let curStages = image.tag ? repo.get(image.tag) : undefined;
        if (!curStages) {
          curStages = {} as StageStyleMap;
        }
        curStages[stage.metadata?.name as string] = {
          opacity: 1,
          backgroundColor: stageColorMap[stage.metadata?.name as string]
        };
        repo.set(image.tag!, curStages);
      });
    });
    return images;
  }, [stages]);
};
