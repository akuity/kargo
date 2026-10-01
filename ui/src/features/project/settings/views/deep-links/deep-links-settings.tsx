import { useParams } from 'react-router-dom';

import { DeepLinks } from '@ui/features/common/settings/deep-links/deep-links';
import { useSaveProjectDeepLinks } from '@ui/features/common/settings/deep-links/use-save-deep-links';
import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';
import { ApiError } from '@ui/lib/api/custom-fetch';

export const DeepLinksSettings = () => {
  const { name = '' } = useParams();

  const getProjectConfigQuery = useGetProjectConfig(name, {
    query: { meta: { silent404: true } }
  });

  const projectConfig = getProjectConfigQuery.data?.data;

  // a missing ProjectConfig is expected -- saving creates it. Any other failure
  // leaves the stored spec unknown, and a save would overwrite all of it
  const { error } = getProjectConfigQuery;
  const loadFailed = !!error && !(error instanceof ApiError && error.isNotFound());

  const { mutateAsync: onUpdate } = useSaveProjectDeepLinks(name);

  return (
    <DeepLinks
      scope='project'
      freightLinks={projectConfig?.spec?.freightLinks ?? []}
      stageLinks={projectConfig?.spec?.stageLinks ?? []}
      loading={getProjectConfigQuery.isLoading}
      loadFailed={loadFailed}
      onUpdate={onUpdate}
    />
  );
};
