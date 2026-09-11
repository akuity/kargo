import {
  faAsterisk,
  faChartBar,
  faGear,
  faGears,
  faKey,
  faScrewdriverWrench,
  faTasks
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu, Skeleton, Typography } from 'antd';
import classNames from 'classnames';
import React from 'react';
import { NavLink, Route, Routes, useLocation, useParams, Navigate } from 'react-router-dom';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useGetConfig } from '@ui/gen/api/v2/system/system';

import { useProjectBreadcrumbs } from '../project-utils';

import { AccessSettings } from './views/access/access-settings';
import { AnalysisTemplatesSettings } from './views/analysis-templates/analysis-templates';
import { ConfigMapsSettings } from './views/config-maps/config-maps-settings';
import { GeneralSettings } from './views/general/general-settings';
import { ProjectConfig } from './views/project-config/project-config';
import { PromotionTasks } from './views/promotion-tasks/promotion-tasks';
import { SecretsSettings } from './views/secrets/secrets-settings';

export const ProjectSettings = () => {
  const location = useLocation();

  const getConfigQuery = useGetConfig();
  const config = getConfigQuery.data?.data;

  const { projectSettingsExtensions, projectConfigSubpages } = useExtensionsContext();

  const settingsViews = React.useMemo(() => {
    return {
      general: {
        label: 'General',
        icon: faGear,
        path: 'general',
        component: GeneralSettings
      },
      projectConfig: {
        label: 'ProjectConfig',
        icon: faGears,
        path: 'project-config',
        component: ProjectConfig
      },
      roles: {
        label: 'Access',
        icon: faKey,
        path: 'access',
        component: AccessSettings
      },
      analysisTemplates: {
        label: 'Analysis Templates',
        icon: faChartBar,
        path: 'analysis-templates',
        component: AnalysisTemplatesSettings
      },
      ...(config?.secretManagementEnabled
        ? {
            credentials: {
              label: 'Secrets',
              icon: faAsterisk,
              path: 'secrets',
              component: SecretsSettings
            }
          }
        : {}),
      configMaps: {
        label: 'ConfigMaps',
        path: 'config-maps',
        icon: faScrewdriverWrench,
        component: ConfigMapsSettings
      },
      promotionTasks: {
        label: 'Promotion Tasks',
        icon: faTasks,
        path: 'promotion-tasks',
        component: PromotionTasks
      }
    };
  }, [config]);

  const configSubpages = React.useMemo(
    () =>
      projectConfigSubpages.map((subpage) => ({
        ...subpage,
        path: `${settingsViews.projectConfig.path}/${subpage.path}`
      })),
    [projectConfigSubpages, settingsViews]
  );

  const views = React.useMemo(
    () => [...Object.values(settingsViews), ...projectSettingsExtensions],
    [projectSettingsExtensions, settingsViews]
  );

  const routableViews = React.useMemo(() => [...views, ...configSubpages], [views, configSubpages]);

  const menuItems = React.useMemo(
    () =>
      views.flatMap((view) => {
        const subpages = view.path === settingsViews.projectConfig.path ? configSubpages : [];
        const onSubpage = subpages.some((subpage) => location.pathname.endsWith(subpage.path));

        return [
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
      }),
    [views, configSubpages, settingsViews, location.pathname]
  );

  const wide = configSubpages.some(
    (subpage) => subpage.wide && location.pathname.endsWith(subpage.path)
  );

  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { name } = useParams();
  useDocumentTitle(['Settings', name]);

  return (
    <>
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[
            ...projectBreadcrumbs,
            {
              title: 'Settings'
            }
          ]}
        />
      </BaseHeader>
      <div className='py-4 px-6'>
        <Typography.Title level={3}>Project Settings</Typography.Title>
        <Flex gap={24} className='mt-2'>
          <div style={{ width: 240 }}>
            <Skeleton loading={getConfigQuery.isFetching} active paragraph={{ rows: 6 }}>
              <Menu
                className='-mt-1'
                mode='inline'
                style={{ border: 0, background: 'transparent' }}
                selectedKeys={routableViews
                  .map((i) => i.path)
                  .filter((i) => location.pathname.endsWith(i))}
                items={menuItems}
              />
            </Skeleton>
          </div>
          <div
            className='flex-1 overflow-hidden'
            style={{ maxWidth: wide ? '1440px' : '920px', minHeight: wide ? undefined : '700px' }}
          >
            <Skeleton loading={getConfigQuery.isFetching} active paragraph={{ rows: 16 }}>
              <Routes>
                <Route
                  index
                  element={<Navigate to={settingsViews.general.path} replace={true} />}
                />
                {routableViews.map((t) => (
                  <Route key={t.path} path={t.path} element={<t.component />} />
                ))}
                <Route path='*' element={<Navigate to='../' replace={true} />} />
              </Routes>
            </Skeleton>
          </div>
        </Flex>
      </div>
    </>
  );
};
