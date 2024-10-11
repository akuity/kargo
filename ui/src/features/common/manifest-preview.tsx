import yaml from 'yaml';

import { Freight, Promotion, Stage } from '@ui/gen/v1alpha1/generated_pb';

import YamlEditor from './code-editor/yaml-editor-lazy';

export const ManifestPreview = ({
  object,
  height = '100%'
}: {
  object: Stage | Freight | Promotion;
  height?: string;
}) => {
  const encodedObject = yaml.stringify(object.toJson(), (_, v) => {
    if (typeof v === 'string' && v === '') {
      return;
    }
    if (Array.isArray(v) && v.length === 0) {
      return;
    }

    // problem: API responds the manifest with nested JSON config as raw Uint8Array JSON string, we just need to convert it
    // happens in promotion directives step YAML view
    if (typeof v.raw === 'string') {
      return JSON.parse(atob(v.raw));
    }
    return v;
  });

  return <YamlEditor value={encodedObject} height={height} disabled isHideManagedFieldsDisplayed />;
};
