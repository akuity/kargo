import { faBarChart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import type { MenuProps } from 'antd';
import { Menu } from 'antd';
import { useNavigate, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
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
      <div className='p-7 float left'>
        <Menu mode='horizontal' items={items} />
      </div>
      {renderSection(section)}
    </div>
  );
};
