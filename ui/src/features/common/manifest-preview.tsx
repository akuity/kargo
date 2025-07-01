import { JsonValue } from '@bufbuild/protobuf';
import yaml from 'yaml';

import YamlEditor from './code-editor/yaml-editor-lazy';

export const ManifestPreview = ({
  object,
  height = '100%'
}: {
  object: JsonValue;
  height?: string;
}) => {
  const encodedObject = yaml.stringify(object, (_, v) => {
    if (!v) {
      return;
    }

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
