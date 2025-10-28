import { Breadcrumb } from 'antd';
import React from 'react';
import { Navigate, useParams } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

import { useExtensionsContext } from '../extensions-context';

export const ArgoCDExtension = () => {
  const { appName } = useParams();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { argoCDExtension } = useExtensionsContext();

  if (!argoCDExtension) {
    return <Navigate to='/' replace />;
  }

  return (
    <>
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[...projectBreadcrumbs, { title: 'ArgoCD' }, { title: appName }]}
        />
        {argoCDExtension.tabBarExtraContent && <argoCDExtension.tabBarExtraContent />}
      </BaseHeader>
      <argoCDExtension.component />
    </>
  );
};
