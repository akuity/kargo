import { Breadcrumb } from 'antd';
import { Navigate, useParams } from 'react-router-dom';

import { LoadingState } from '@ui/features/common';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

import { useExtensionsContext } from '../extensions-context';
import { useIsAnyExtensionLoaded } from '../utils';

export const ArgoCDExtension = () => {
  const { appName } = useParams();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { argoCDExtension } = useExtensionsContext();
  const isAnyExenstionLoaded = useIsAnyExtensionLoaded();

  if (!isAnyExenstionLoaded) {
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
      {argoCDExtension ? <argoCDExtension.component /> : <LoadingState />}
    </>
  );
};
