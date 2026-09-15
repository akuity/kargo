import {
  faAsterisk,
  faBarChart,
  faDisplay,
  faGear,
  faKey,
  faScrewdriverWrench,
  faTasks
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu } from 'antd';
import { ItemType, MenuItemType } from 'antd/es/menu/interface';
import classNames from 'classnames';
import React from 'react';
import { NavLink, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { AccessSettings } from '@ui/features/settings/access/accecss';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';
import { ClusterConfig } from '@ui/features/settings/cluster-config/cluster-config';
import { ClusterPromotionTasks } from '@ui/features/settings/cluster-promotion-tasks/cluster-promotion-tasks';
import { ClusterSecret } from '@ui/features/settings/cluster-secret/cluster-secret';
import { ConfigMapsSettings } from '@ui/features/settings/config-maps/config-maps-settings';
import { SharedSecrets } from '@ui/features/settings/shared-secrets/shared-secrets';
import { UISettings } from '@ui/features/settings/ui/ui-settings';

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
  },
  ui: {
    label: 'UI Preferences',
    icon: faDisplay,
    path: 'ui',
    component: UISettings
  }
};

const defaultView = settingsViews.clusterConfig;

export const Settings = () => {
  useDocumentTitle(['Settings']);
  const location = useLocation();
  const { settingsExtensions, clusterConfigSubpages } = useExtensionsContext();

  const configSubpages = React.useMemo(
    () =>
      clusterConfigSubpages.map((subpage) => ({
        ...subpage,
        path: `${settingsViews.clusterConfig.path}/${subpage.path}`
      })),
    [clusterConfigSubpages]
  );

  const views = React.useMemo(
    () => [...Object.values(settingsViews), ...settingsExtensions],
    [settingsExtensions]
  );

  const routableViews = React.useMemo(() => [...views, ...configSubpages], [views, configSubpages]);

  const wide = configSubpages.some(
    (subpage) => subpage.wide && location.pathname.endsWith(subpage.path)
  );

  const menuItems = React.useMemo(
    () =>
      views.reduce((acc, view) => {
        const group = ('group' in view ? view.group : DEFAULT_GROUP) as string;
        const groupIndex = acc.findIndex((g) => g?.key === group);

        const subpages = view.path === settingsViews.clusterConfig.path ? configSubpages : [];

        const onSubpage = subpages.some((subpage) => location.pathname.endsWith(subpage.path));

        const items = [
          {
            label: (
              <NavLink to={`../${view.path}`} style={{ color: 'inherit' }}>
                {view.label}
              </NavLink>
            ),
            icon: <FontAwesomeIcon icon={view.icon} />,
            key: view.path,
            // The `!`s beat AntD's own rules for these properties.
            className: classNames('!pl-3', {
              '!text-[var(--kargo-color-text-base)]': onSubpage
            })
          },
          ...subpages.map((subpage) => ({
            label: (
              <NavLink to={`../${subpage.path}`} style={{ color: 'inherit' }}>
                {subpage.label}
              </NavLink>
            ),
            key: subpage.path,
            // `overflow-visible` keeps AntD from clipping the guide line;
            // labels still ellipsize via its rule on `.ant-menu-title-content`.
            className: classNames(
              'relative !overflow-visible !ms-8 !w-auto !pl-2',
              "before:absolute before:content-[''] before:-left-[9px] before:-top-1",
              'before:-bottom-1 before:w-px before:bg-[var(--kargo-color-border)]'
            )
          }))
        ];

        if (groupIndex === -1) {
          acc.push({ key: group, label: group, type: 'group', children: items });
        } else if (acc[groupIndex] && 'children' in acc[groupIndex]) {
          acc[groupIndex].children?.push(...items);
        }

        return acc;
      }, [] as ItemType<MenuItemType>[]),
    [views, configSubpages, location.pathname]
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
              className='-mt-1 mb-4'
              mode='inline'
              style={{ border: 0, background: 'transparent' }}
              selectedKeys={routableViews
                .map((i) => i.path)
                .filter((i) => location.pathname.endsWith(i))}
              items={menuItems}
            />
          </div>
          <div
            className='flex-1 overflow-hidden'
            style={{ maxWidth: wide ? '1440px' : '920px', minHeight: wide ? undefined : '700px' }}
          >
            <Routes>
              <Route index element={<Navigate to={defaultView.path} replace={true} />} />
              {routableViews.map((t) => (
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
