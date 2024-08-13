import { useQuery } from '@connectrpc/connect-query';
import {
  faChartBar,
  faClockRotateLeft,
  faDiagramProject,
  faIdBadge,
  faPeopleGroup
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { SmallLabel } from '@ui/features/common/small-label';
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

export const Project = ({
  tab = 'pipelines',
  creatingStage
}: {
  tab?: ProjectTab;
  creatingStage?: boolean;
}) => {
  const { name } = useParams();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery(getProject, { name });

  // we must render the tab contents outside of the Antd tabs component to prevent layout issues in the ProjectDetails component
  const renderTab = (key: ProjectTab) => {
    switch (key) {
      case 'pipelines':
        return (
          <Pipelines project={data?.result?.value as _Project} creatingStage={creatingStage} />
        );
      case 'credentials':
        return <CredentialsList />;
      case 'analysisTemplates':
        return <AnalysisTemplatesList />;
      case 'events':
        return <Events />;
      case 'roles':
        return <Roles />;
      default:
        return <Pipelines project={data?.result?.value as _Project} />;
    }
  };

  return (
    <div className='h-full flex flex-col'>
      <div className='px-6 pt-5 pb-3 mb-2'>
        <div className='flex items-center'>
          <div className='mr-auto'>
            <SmallLabel>PROJECT</SmallLabel>
            <div className='text-2xl font-semibold flex items-center'>
              {name} <ProjectSettings />
            </div>
            <Description
              loading={isLoading}
              item={data?.result?.value as _Project}
              className='mt-1'
            />
          </div>
          <div className='flex items-center gap-8 text-gray-500 text-sm mr-2'>
            {Object.entries(tabs).map(([key, value]) => (
              <Tooltip key={key} title={value.label}>
                <div className={classNames('cursor-pointer', { 'text-blue-500': tab === key })}>
                  <FontAwesomeIcon
                    icon={value.icon}
                    onClick={() => {
                      navigate(generatePath(tabs[key as ProjectTab].path, { name }));
                    }}
                  />
                </div>
              </Tooltip>
            ))}
          </div>
        </div>
      </div>
      {renderTab(tab)}
    </div>
  );
};
