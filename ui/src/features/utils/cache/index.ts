// cache invalidation source-of-truth

import analysisTemplates from './analysis-templates';
import imageStageMatrix from './image-stage-matrix';
import project from './project';

export const queryCache = {
  project,
  analysisTemplates,
  imageStageMatrix
};
