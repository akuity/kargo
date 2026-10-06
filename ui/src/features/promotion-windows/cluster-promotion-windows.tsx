import { Skeleton } from 'antd';

import { useGetClusterConfig } from '@ui/gen/api/v2/system/system';

import { PromotionWindows } from './promotion-windows';
import { useSaveClusterPromotionWindows } from './use-save-promotion-windows';

export const ClusterPromotionWindows = () => {
  const getClusterConfigQuery = useGetClusterConfig({ query: { meta: { silent404: true } } });

  const clusterConfig = getClusterConfigQuery.data?.data;

  const saveMutation = useSaveClusterPromotionWindows(clusterConfig);

  return (
    <Skeleton loading={getClusterConfigQuery.isLoading} active paragraph={{ rows: 12 }}>
      <PromotionWindows
        scope='cluster'
        promotionWindows={clusterConfig?.spec?.promotionWindows ?? []}
        onUpdate={saveMutation.mutate}
      />
    </Skeleton>
  );
};
