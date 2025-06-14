// cache invalidation source-of-truth

import analysisTemplates from './analysis-templates';
import freight from './freight';
import imageStageMatrix from './image-stage-matrix';
import project from './project';
import projectConfig from './project-config';

export const queryCache = {
  project,
  analysisTemplates,
  imageStageMatrix,
  freight,
  projectConfig
};
