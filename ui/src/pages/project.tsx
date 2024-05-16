import { useQuery } from '@connectrpc/connect-query';
import {
  faChartBar,
  faClockRotateLeft,
  faDiagramProject,
  faIdBadge,
  faPeopleGroup
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tabs } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { AnalysisTemplatesList } from '@ui/features/project/analysis-templates/analysis-templates-list';
import { CredentialsList } from '@ui/features/project/credentials/credentials-list';
import { Events } from '@ui/features/project/events/events';
import { Pipelines } from '@ui/features/project/pipelines/pipelines';
import { Roles } from '@ui/features/project/roles/roles';
import { ProjectSettings } from '@ui/features/project/settings/project-settings';
import { getProject } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Project as _Project } from '@ui/gen/v1alpha1/generated_pb';

const tabs = {
  pipelines: {
    path: paths.project,
    label: 'Pipelines',
    icon: faDiagramProject
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
  },
  events: {
    path: paths.projectEvents,
    label: 'Events',
    icon: faClockRotateLeft
  },
  roles: {
    path: paths.projectRoles,
    label: 'Roles',
    icon: faPeopleGroup
  }
};

export type ProjectTab = keyof typeof tabs;

export const Project = ({ tab = 'pipelines' }: { tab?: ProjectTab }) => {
  const { name } = useParams();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery(getProject, { name });

  // we must render the tab contents outside of the Antd tabs component to prevent layout issues in the ProjectDetails component
  const renderTab = (key: ProjectTab) => {
    switch (key) {
      case 'pipelines':
        return <Pipelines />;
      case 'credentials':
        return <CredentialsList />;
      case 'analysisTemplates':
        return <AnalysisTemplatesList />;
      case 'events':
        return <Events />;
      case 'roles':
        return <Roles />;
      default:
        return <Pipelines />;
    }
  };

  return (
    <div className='h-full flex flex-col'>
      <div className='px-6 pt-5 pb-3'>
        <div className='flex items-center'>
          <div className='mr-auto'>
            <div className='font-medium text-xs text-neutral-500'>PROJECT</div>
            <div className='text-2xl font-semibold'>{name}</div>
            <Description
              loading={isLoading}
              item={data?.result?.value as _Project}
              className='mt-1'
            />
          </div>
          <ProjectSettings />
        </div>
      </div>
      <Tabs
        activeKey={tab}
        onChange={(k) => {
          navigate(generatePath(tabs[k as ProjectTab].path, { name }));
        }}
        tabBarStyle={{
          padding: '0 24px',
          marginBottom: '0.25rem'
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
