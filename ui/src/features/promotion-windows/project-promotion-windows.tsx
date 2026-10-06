import { Skeleton } from 'antd';
import { useParams } from 'react-router-dom';

import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';

import { PromotionWindows } from './promotion-windows';
import { useSaveProjectPromotionWindows } from './use-save-promotion-windows';

export const ProjectPromotionWindows = () => {
  const { name = '' } = useParams();

  const getProjectConfigQuery = useGetProjectConfig(name, {
    query: { meta: { silent404: true } }
  });

  const projectConfig = getProjectConfigQuery.data?.data;

  const saveMutation = useSaveProjectPromotionWindows(name, projectConfig);

  return (
    <Skeleton loading={getProjectConfigQuery.isLoading} active paragraph={{ rows: 12 }}>
      <PromotionWindows
        scope='project'
        promotionWindows={projectConfig?.spec?.promotionWindows ?? []}
        onUpdate={saveMutation.mutate}
      />
    </Skeleton>
  );
};
