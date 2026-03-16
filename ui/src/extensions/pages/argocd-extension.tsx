import { useQuery } from '@connectrpc/connect-query';
import { Breadcrumb } from 'antd';
import { Navigate, useParams } from 'react-router-dom';
import yaml from 'yaml';

import { SHARD_LABEL_KEY } from '@ui/config/labels';
import { LoadingState } from '@ui/features/common';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';
import {
  getConfig,
  getStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { useExtensionsContext } from '../extensions-context';
import { useIsAnyExtensionLoaded } from '../utils';

export const ArgoCDExtension = () => {
  const { appName, stageName, namespace, name } = useParams();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { argoCDExtension } = useExtensionsContext();
  const isAnyExenstionLoaded = useIsAnyExtensionLoaded();

  const hasParams = !!namespace && !!stageName;

  const stageQuery = useQuery(
    getStage,
    { project: name, name: stageName, format: RawFormat.YAML },
    { enabled: hasParams }
  );
  const configQuery = useQuery(getConfig, {}, { enabled: hasParams });

  if (!isAnyExenstionLoaded || !appName || !stageName || !namespace) {
    return <Navigate to='/' replace />;
  }

  const isLoading = stageQuery.isPending || configQuery.isPending;

  let shardKey = '';

  try {
    shardKey =
      yaml.parse(decodeRawData(stageQuery.data))?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  } catch {
    shardKey = '';
  }
  const argocdURL = configQuery.data?.argocdShards?.[shardKey]?.url?.replace(/\/$/, '') || '';

  return (
    <>
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[...projectBreadcrumbs, { title: 'ArgoCD' }, { title: appName }]}
        />
      </BaseHeader>
      {isLoading || !argoCDExtension ? (
        <LoadingState />
      ) : (
        <argoCDExtension.component
          stageName={stageName}
          namespace={namespace}
          appName={appName}
          argocdURL={argocdURL}
        />
      )}
    </>
  );
};
