import { useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

import { StageStyleMap } from '../types';

export const useImages = (stages: Stage[]) => {
  const colors = useContext(ColorContext);

  return useMemo(() => {
    const images = new Map<string, Map<string, StageStyleMap>>();
    stages.forEach((stage) => {
      const len = stage.status?.history?.length || 0;
      stage.status?.history?.forEach((freight, i) => {
        freight.images?.forEach((image) => {
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
            backgroundColor: colors[stage.metadata?.name as string]
          };
          repo.set(image.tag!, curStages);
        });
      });

      stage.status?.currentFreight?.images?.forEach((image) => {
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
          backgroundColor: colors[stage.metadata?.name as string]
        };
        repo.set(image.tag!, curStages);
      });
    });
    return images;
  }, [stages]);
};
