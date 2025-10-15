import { faAsterisk, faBarChart, faGear, faTasks } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu, Typography } from 'antd';
import React from 'react';
import { NavLink, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';
import { ClusterConfig } from '@ui/features/settings/cluster-config/cluster-config';
import { ClusterPromotionTasks } from '@ui/features/settings/cluster-promotion-tasks/cluster-promotion-tasks';
import { ClusterSecret } from '@ui/features/settings/cluster-secret/cluster-secret';

const settingsViews = {
  clusterConfig: {
    label: 'Cluster Config',
    icon: faGear,
    path: 'cluster-config',
    component: ClusterConfig
  },
  verification: {
    label: 'Verification',
    icon: faBarChart,
    path: 'analysis-templates',
    component: ClusterAnalysisTemplatesList
  },
  clusterPromotionTasks: {
    label: 'ClusterPromotionTasks',
    icon: faTasks,
    path: 'cluster-promotion-tasks',
    component: ClusterPromotionTasks
  },
  clusterSecret: {
    label: 'Cluster Secrets',
    icon: faAsterisk,
    component: ClusterSecret,
    path: 'cluster-secrets'
  }
};

const defaultView = settingsViews.clusterConfig;

export const Settings = () => {
  const location = useLocation();
  const { settingsExtensions } = useExtensionsContext();

  const views = React.useMemo(
    () => [...Object.values(settingsViews), ...settingsExtensions],
    [settingsExtensions]
  );

  return (
    <>
      <BaseHeader>
        <Breadcrumb separator='>' items={[{ title: 'Settings' }]} />
      </BaseHeader>
      <div className='py-4 px-6'>
        <Typography.Title level={3}>Settings</Typography.Title>
        <Flex gap={24} className='mt-2'>
          <div style={{ width: 240 }}>
            <Menu
              className='-ml-2 -mt-1'
              style={{ border: 0, background: 'transparent' }}
              selectedKeys={views.map((i) => i.path).filter((i) => location.pathname.endsWith(i))}
              items={views.map((i) => ({
                label: <NavLink to={`../${i.path}`}>{i.label}</NavLink>,
                icon: <FontAwesomeIcon icon={i.icon} />,
                key: i.path
              }))}
            />
          </div>
          <div className='flex-1 overflow-hidden' style={{ maxWidth: '920px', minHeight: '700px' }}>
            <Routes>
              <Route index element={<Navigate to={defaultView.path} replace={true} />} />
              {views.map((t) => (
                <Route key={t.path} path={t.path} element={<t.component />} />
              ))}
              <Route path='*' element={<Navigate to='../' replace={true} />} />
            </Routes>
          </div>
        </Flex>
      </div>
    </>
  );
};
