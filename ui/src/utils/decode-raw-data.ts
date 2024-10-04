import yaml from 'yaml';

type Data = {
  result:
    | {
        value: unknown;
        case: 'stage' | 'project' | 'analysisRun' | 'analysisTemplate' | 'warehouse';
      }
    | {
        value: Uint8Array;
        case: 'raw';
      }
    | { case: undefined; value?: undefined };
};

export const decodeRawData = (data?: Data) =>
  new TextDecoder().decode(
    data?.result?.case === 'raw' ? (data?.result?.value ?? new Uint8Array()) : new Uint8Array()
  );

export const decodeUint8ArrayYamlManifestToJson = <T>(raw: Uint8Array): T => {
  return yaml.parse(new TextDecoder().decode(raw));
};
