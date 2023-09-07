import yaml from 'yaml';

import { Stage } from '@ui/gen/v1alpha1/types_pb';

export const prepareStageToEdit = (stage?: Stage) => {
  if (!stage) return '';

  return yaml.stringify({
    metadata: {
      annotations: stage.metadata?.annotations || {},
      labels: stage.metadata?.labels || {}
    },
    spec: stage.spec
  });
};

export const prepareStageToSave = (stage: Stage | undefined, updatedStage: string) => {
  if (!stage) return '';

  const data = yaml.parse(updatedStage);

  return yaml.stringify({
    ...stage,
    ...data,
    metadata: {
      ...stage.metadata,
      ...data.metadata
    }
  });
};
