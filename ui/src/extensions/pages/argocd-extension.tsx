import { Breadcrumb } from 'antd';
import { Navigate, useParams } from 'react-router-dom';

import { LoadingState } from '@ui/features/common';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

import { useExtensionsContext } from '../extensions-context';
import { useIsAnyExtensionLoaded } from '../utils';

import { useArgoCDURL } from './use-argocd-url';

export const ArgoCDExtension = () => {
  const { appName, stageName, namespace, name } = useParams();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { argoCDExtension } = useExtensionsContext();
  const isAnyExenstionLoaded = useIsAnyExtensionLoaded();

  const { argocdURL, isPending } = useArgoCDURL({ project: name, stageName });

  if (!isAnyExenstionLoaded || !appName || !stageName || !namespace) {
    return <Navigate to='/' replace />;
  }

  return (
    <>
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[...projectBreadcrumbs, { title: 'ArgoCD' }, { title: appName }]}
        />
      </BaseHeader>
      {isPending || !argoCDExtension ? (
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
