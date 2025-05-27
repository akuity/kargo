import { useQuery } from '@connectrpc/connect-query';
import {
  faAsterisk,
  faChartBar,
  faGear,
  faGears,
  faPeopleGroup,
  faTasks
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu, Skeleton, Typography } from 'antd';
import React from 'react';
import { NavLink, Route, Routes, useLocation, Navigate } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';
import { getConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { useProjectBreadcrumbs } from '../project-utils';

import { AnalysisTemplatesSettings } from './views/analysis-templates/analysis-templates';
import { CredentialsSettings } from './views/credentials/credentials';
import { GeneralSettings } from './views/general/general-settings';
import { ProjectConfig } from './views/project-config/project-config';
import { PromotionTasks } from './views/promotion-tasks/promotion-tasks';
import { RolesSettings } from './views/roles/roles';

export const ProjectSettings = () => {
  const location = useLocation();

  const { data: config, isFetching } = useQuery(getConfig);

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
              component: CredentialsSettings
            }
          }
        : {}),
      roles: {
        label: 'Roles',
        icon: faPeopleGroup,
        path: 'roles',
        component: RolesSettings
      },
      promotionTasks: {
        label: 'Promotion Tasks',
        icon: faTasks,
        path: 'promotion-tasks',
        component: PromotionTasks
      }
    };
  }, [config]);

  const projectBreadcrumbs = useProjectBreadcrumbs();

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
            <Skeleton loading={isFetching} active paragraph={{ rows: 6 }}>
              <Menu
                className='-ml-2 -mt-1'
                style={{ border: 0, background: 'transparent' }}
                selectedKeys={Object.values(settingsViews)
                  .map((i) => i.path)
                  .filter((i) => location.pathname.endsWith(i))}
                items={Object.values(settingsViews).map((i) => ({
                  label: <NavLink to={`../${i.path}`}>{i.label}</NavLink>,
                  icon: <FontAwesomeIcon icon={i.icon} />,
                  key: i.path
                }))}
              />
            </Skeleton>
          </div>
          <div className='flex-1 overflow-hidden' style={{ maxWidth: '920px', minHeight: '700px' }}>
            <Skeleton loading={isFetching} active paragraph={{ rows: 16 }}>
              <Routes>
                <Route
                  index
                  element={<Navigate to={settingsViews.general.path} replace={true} />}
                />
                {Object.values(settingsViews).map((t) => (
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
