type Data = {
  result:
    | {
        value: unknown;
        case:
          | 'stage'
          | 'project'
          | 'analysisRun'
          | 'analysisTemplate'
          | 'clusterAnalysisTemplate'
          | 'warehouse';
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
