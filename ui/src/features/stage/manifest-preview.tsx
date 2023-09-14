import yaml from 'yaml';

import { Stage } from '@ui/gen/v1alpha1/types_pb';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';

export const ManifestPreview = ({ stage }: { stage: Stage }) => {
  const encodedStage = yaml.stringify(stage.toJson(), (_, v) => {
    if (typeof v === 'string' && v === '') {
      return;
    }
    if (Array.isArray(v) && v.length === 0) {
      return;
    }
    return v;
  });
  return <YamlEditor value={encodedStage} height='500px' disabled />;
};
