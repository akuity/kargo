import yaml from 'yaml';

import { Stage } from '@ui/gen/v1alpha1/types_pb';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';

export const ManifestPreview = ({ stage }: { stage: Stage }) => {
  return <YamlEditor value={yaml.stringify(stage)} height='500px' disabled />;
};
