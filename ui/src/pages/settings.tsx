import {
  faAsterisk,
  faBarChart,
  faGear,
  faKey,
  faScrewdriverWrench,
  faTasks
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu } from 'antd';
import { ItemType, MenuItemType } from 'antd/es/menu/interface';
import React from 'react';
import { NavLink, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { AccessSettings } from '@ui/features/settings/access/accecss';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';
import { ClusterConfig } from '@ui/features/settings/cluster-config/cluster-config';
import { ClusterPromotionTasks } from '@ui/features/settings/cluster-promotion-tasks/cluster-promotion-tasks';
import { ClusterSecret } from '@ui/features/settings/cluster-secret/cluster-secret';
import { ConfigMapsSettings } from '@ui/features/settings/config-maps/config-maps-settings';
import { SharedSecrets } from '@ui/features/settings/shared-secrets/shared-secrets';

const DEFAULT_GROUP = 'General';

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
    component: ClusterAnalysisTemplatesList,
    group: 'Projects'
  },
  clusterSecret: {
    label: 'System Secrets',
    icon: faAsterisk,
    component: ClusterSecret,
    path: 'system-secrets'
  },
  access: {
    label: 'Access',
    icon: faKey,
    component: AccessSettings,
    path: 'access'
  },
  sharedSecret: {
    label: 'Secrets',
    icon: faAsterisk,
    component: SharedSecrets,
    path: 'shared-secrets',
    group: 'Projects'
  },
  configMaps: {
    label: 'ConfigMaps',
    icon: faScrewdriverWrench,
    component: ConfigMapsSettings,
    path: 'config-maps',
    group: 'Projects'
  },
  clusterPromotionTasks: {
    label: 'ClusterPromotionTasks',
    icon: faTasks,
    path: 'cluster-promotion-tasks',
    component: ClusterPromotionTasks,
    group: 'Projects'
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

  const menuItems = React.useMemo(
    () =>
      views.reduce((acc, view) => {
        const group = ('group' in view ? view.group : DEFAULT_GROUP) as string;
        const groupIndex = acc.findIndex((g) => g?.key === group);

        const children = {
          label: <NavLink to={`../${view.path}`}>{view.label}</NavLink>,
          icon: <FontAwesomeIcon icon={view.icon} />,
          key: view.path
        };

        if (groupIndex === -1) {
          acc.push({ key: group, label: group, type: 'group', children: [children] });
        } else if (acc[groupIndex] && 'children' in acc[groupIndex]) {
          acc[groupIndex].children?.push(children);
        }

        return acc;
      }, [] as ItemType<MenuItemType>[]),
    [views]
  );

  return (
    <>
      <BaseHeader>
        <Breadcrumb separator='>' items={[{ title: 'Settings' }]} />
      </BaseHeader>
      <div className='py-4 px-6'>
        <Flex gap={24} className='mt-2'>
          <div style={{ width: 240 }}>
            <Menu
              className='-ml-2 -mt-1 mb-4'
              style={{ border: 0, background: 'transparent' }}
              selectedKeys={views.map((i) => i.path).filter((i) => location.pathname.endsWith(i))}
              items={menuItems}
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
