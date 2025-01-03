import { faBarChart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Layout, Menu, MenuProps } from 'antd';
import { useNavigate, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PageTitle } from '@ui/features/common';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';

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
    }
  ];

  const renderSection = (section: string) => {
    switch (section) {
      case 'verification':
        return <ClusterAnalysisTemplatesList />;
    }
  };

  const getSectionTitle = (section: string) => {
    switch (section) {
      case 'verification':
        return 'Cluster Analysis Templates';
    }
  };

  return (
    <Layout className='min-h-screen'>
      <Sider width={250} className='bg-white border-r border-gray-300 shadow-sm'>
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
