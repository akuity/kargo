import { faBarChart, faTasks } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Flex, Menu, Typography } from 'antd';
import { NavLink, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';
import { ClusterPromotionTasks } from '@ui/features/settings/cluster-promotion-tasks/cluster-promotion-tasks';

const settingsViews = {
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
  }
};

const defaultView = settingsViews.verification;

export const Settings = () => {
  const location = useLocation();

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
              selectedKeys={Object.values(settingsViews)
                .map((i) => i.path)
                .filter((i) => location.pathname.endsWith(i))}
              items={Object.values(settingsViews).map((i) => ({
                label: <NavLink to={`../${i.path}`}>{i.label}</NavLink>,
                icon: <FontAwesomeIcon icon={i.icon} />,
                key: i.path
              }))}
            />
          </div>
          <div className='flex-1 overflow-hidden' style={{ maxWidth: '920px', minHeight: '700px' }}>
            <Routes>
              <Route index element={<Navigate to={defaultView.path} replace={true} />} />
              {Object.values(settingsViews).map((t) => (
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
