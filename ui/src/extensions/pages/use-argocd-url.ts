import { SHARD_LABEL_KEY } from '@ui/config/labels';
import { useGetStage } from '@ui/gen/api/v2/core/core';
import { useGetConfig } from '@ui/gen/api/v2/system/system';

export const useArgoCDURL = (params: {
  project: string | undefined;
  stageName: string | undefined;
}) => {
  const enabled = !!params.project && !!params.stageName;

  const stageQuery = useGetStage(params.project || '', params.stageName || '', {
    query: { enabled }
  });
  const configQuery = useGetConfig({ query: { enabled } });

  const shardKey = stageQuery.data?.data?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  const argocdURL = configQuery.data?.data?.argocdShards?.[shardKey]?.url?.replace(/\/$/, '') || '';

  return {
    argocdURL,
    isPending: stageQuery.isPending || configQuery.isPending
  };
};
