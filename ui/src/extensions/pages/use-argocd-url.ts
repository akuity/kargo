import { useQuery } from '@connectrpc/connect-query';
import yaml from 'yaml';

import { SHARD_LABEL_KEY } from '@ui/config/labels';
import {
  getConfig,
  getStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

export const useArgoCDURL = (params: {
  project: string | undefined;
  stageName: string | undefined;
}) => {
  const enabled = !!params.project && !!params.stageName;

  const stageQuery = useQuery(
    getStage,
    { project: params.project, name: params.stageName, format: RawFormat.YAML },
    { enabled }
  );
  const configQuery = useQuery(getConfig, {}, { enabled });

  let shardKey = '';
  try {
    shardKey =
      yaml.parse(decodeRawData(stageQuery.data))?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  } catch {
    shardKey = '';
  }

  const argocdURL = configQuery.data?.argocdShards?.[shardKey]?.url?.replace(/\/$/, '') || '';

  return {
    argocdURL,
    isPending: stageQuery.isPending || configQuery.isPending
  };
};
