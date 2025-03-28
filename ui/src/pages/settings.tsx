import { faBarChart, faTasks } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Layout, Menu, MenuProps } from 'antd';
import { useNavigate, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PageTitle } from '@ui/features/common';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';
import { ClusterPromotionTasks } from '@ui/features/settings/cluster-promotion-tasks/cluster-promotion-tasks';

const { Sider, Content } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

export const Settings = ({ section = 'verification' }: { section?: string }) => {
  const navigate = useNavigate();

  const items: MenuItem[] = [
    {
      key: 'verification',
      label: 'Verification',
      icon: <FontAwesomeIcon icon={faBarChart} />,
      onClick: () => {
        navigate(generatePath(paths.settingsAnalysisTemplates));
      }
    },
    {
      key: 'cluster-promotion-tasks',
      label: 'ClusterPromotionTasks',
      icon: <FontAwesomeIcon icon={faTasks} />,
      onClick: () => {
        navigate(generatePath(paths.settingsClusterPromotionTasks));
      }
    }
  ];

  const renderSection = (section: string) => {
    switch (section) {
      case 'verification':
        return <ClusterAnalysisTemplatesList />;
      case 'cluster-promotion-tasks':
        return <ClusterPromotionTasks />;
    }
  };

  const getSectionTitle = (section: string) => {
    switch (section) {
      case 'verification':
        return 'Cluster Analysis Templates';
      case 'cluster-promotion-tasks':
        return 'Cluster Promotion Tasks';
    }
  };

  return (
    <Layout className='min-h-screen'>
      <Sider
        width={250}
        className='border-r border-gray-300 shadow-sm'
        style={{ background: 'white' }}
      >
        <div className='p-4'>
          <PageTitle title='Settings' />
        </div>
        <Menu mode='vertical' defaultSelectedKeys={[section]} items={items} />
      </Sider>
      <Layout>
        <Content className='p-6 bg-gray-100'>
          <div className='text-2xl font-semibold mb-4'>{getSectionTitle(section)}</div>
          {renderSection(section)}
        </Content>
      </Layout>
    </Layout>
  );
};
