import { ImageStageMap, TagMap } from '@ui/gen/api/service/v1alpha1/service_pb';

import { ProcessedTagMap } from './types';

const filterStagesForImage = (
  imageStageMap: ImageStageMap,
  stagesForImage: Set<string>
): Record<string, number> => {
  const filteredStages: Record<string, number> = {};
  for (const [stageName, order] of Object.entries(imageStageMap.stages || {})) {
    if (stagesForImage.has(stageName)) {
      filteredStages[stageName] = order;
    }
  }
  return filteredStages;
};

export const processTagMap = (tagMap: TagMap, stagesForImage: Set<string>): ProcessedTagMap => {
  const filteredTagMap: ProcessedTagMap = { tags: {} };

  for (const [tag, imageStageMap] of Object.entries(tagMap.tags || {})) {
    const filteredStages = filterStagesForImage(imageStageMap, stagesForImage);
    if (Object.keys(filteredStages).length > 0) {
      filteredTagMap.tags[tag] = { ...imageStageMap, stages: filteredStages };
    }
  }

  return filteredTagMap;
};
