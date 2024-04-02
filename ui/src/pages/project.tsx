import { faChartBar, faCircleNodes, faIdBadge } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tabs } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { AnalysisTemplatesList } from '@ui/features/project/analysis-templates/analysis-templates-list';
import { CredentialsList } from '@ui/features/project/credentials/credentials-list';
import { ProjectDetails } from '@ui/features/project/project-details/project-details';

const tabs = {
  details: {
    path: paths.project,
    label: 'Details',
    icon: faCircleNodes
  },
  credentials: {
    path: paths.projectCredentials,
    label: 'Credentials',
    icon: faIdBadge
  },
  analysisTemplates: {
    path: paths.projectAnalysisTemplates,
    label: 'Analysis Templates',
    icon: faChartBar
  }
};

export type ProjectTab = keyof typeof tabs;

export const Project = ({ tab = 'details' }: { tab?: ProjectTab }) => {
  const { name } = useParams();
  const navigate = useNavigate();

  // we must render the tab contents outside of the Antd tabs component to prevent layout issues in the ProjectDetails component
  const renderTab = (key: ProjectTab) => {
    switch (key) {
      case 'details':
        return <ProjectDetails />;
      case 'credentials':
        return <CredentialsList />;
      case 'analysisTemplates':
        return <AnalysisTemplatesList />;
      default:
        return <ProjectDetails />;
    }
  };

  return (
    <div className='h-full flex flex-col'>
      <div className='p-6'>
        <div className='flex items-center'>
          <div className='mr-auto'>
            <div className='font-semibold mb-1 text-xs text-gray-600'>PROJECT</div>
            <div className='text-2xl font-semibold'>{name}</div>
          </div>
        </div>
      </div>
      <Tabs
        activeKey={tab}
        onChange={(k) => {
          navigate(generatePath(tabs[k as ProjectTab].path, { name }));
        }}
        tabBarStyle={{
          padding: '0 24px',
          marginBottom: '0.5rem'
        }}
        items={Object.entries(tabs).map(([key, value]) => ({
          key,
          label: value.label,
          icon: <FontAwesomeIcon icon={value.icon} />
        }))}
      />
      {renderTab(tab)}
    </div>
  );
};
