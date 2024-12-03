import { faBarChart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import type { MenuProps } from 'antd';
import { Menu, Flex } from 'antd';
import { useNavigate, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PageTitle } from '@ui/features/common';
import { ClusterAnalysisTemplatesList } from '@ui/features/settings/analysis-templates/analysis-templates';

export const Settings = ({ section = 'verification' }: { section?: string }) => {
  const navigate = useNavigate();
  type MenuItem = Required<MenuProps>['items'][number];

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

  return (
    <div className='p-6'>
      <Flex justify='space-between'>
        <PageTitle title='Settings' />
        <div className='text-2xl font-semibold flex items-center'>Cluster Analysis Templates</div>
      </Flex>
      <Flex justify='space-between'>
        <Menu mode='vertical' items={items} />
        {renderSection(section)}
      </Flex>
    </div>
  );
};
