import { useQuery } from '@connectrpc/connect-query';
import {
  faAsterisk,
  faChartBar,
  faClockRotateLeft,
  faDiagramProject,
  faPeopleGroup,
  faTasks
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Result, Tooltip } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { Description } from '@ui/features/common/description';
import { SmallLabel } from '@ui/features/common/small-label';
import { AnalysisTemplatesList } from '@ui/features/project/analysis-templates/analysis-templates-list';
import { CredentialsList } from '@ui/features/project/credentials/credentials-list';
import { Events } from '@ui/features/project/events/events';
import { Pipelines } from '@ui/features/project/pipelines/pipelines';
import { Roles } from '@ui/features/project/roles/roles';
import { ProjectSettings } from '@ui/features/project/settings/project-settings';
import { PromotionTasks } from '@ui/features/promotion-tasks/promotion-tasks';
import {
  getConfig,
  getProject
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Project as _Project } from '@ui/gen/api/v1alpha1/generated_pb';

export const Project = ({
  tab = 'pipelines',
  creatingStage,
  creatingWarehouse
}: {
  tab?: string;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name } = useParams();
  const navigate = useNavigate();

  const { data, isLoading, error } = useQuery(getProject, { name });
  const { data: config } = useQuery(getConfig);

  const [tabs] = useMemo(() => {
    return [
      {
        pipelines: {
          path: paths.project,
          label: 'Pipelines',
          icon: faDiagramProject
        },
        ...(config?.secretManagementEnabled
          ? {
              credentials: {
                path: paths.projectCredentials,
                label: 'Secrets',
                icon: faAsterisk
              }
            }
          : {}),
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
        },
        promotionTasks: {
          path: paths.promotionTasks,
          label: 'Promotion Tasks',
          icon: faTasks
        }
      }
    ];
  }, [config]);

  if (isLoading) {
    return <LoadingState />;
  }

  if (error) {
    return (
      <Result
        status='404'
        title='Error'
        subTitle={error?.message}
        extra={
          <Button type='primary' onClick={() => navigate(paths.projects)}>
            Go to Projects Page
          </Button>
        }
      />
    );
  }

  // we must render the tab contents outside of the Antd tabs component to prevent layout issues in the ProjectDetails component
  const renderTab = (key: string) => {
    switch (key) {
      case 'pipelines':
        return (
          <Pipelines
            project={data?.result?.value as _Project}
            creatingStage={creatingStage}
            creatingWarehouse={creatingWarehouse}
          />
        );
      case 'credentials':
        return config?.secretManagementEnabled ? (
          <CredentialsList />
        ) : (
          <Pipelines project={data?.result?.value as _Project} />
        );
      case 'analysisTemplates':
        return <AnalysisTemplatesList />;
      case 'events':
        return <Events />;
      case 'roles':
        return <Roles />;
      case 'promotionTasks':
        return <PromotionTasks />;
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
                <div
                  className={classNames('cursor-pointer', { 'text-blue-500': tab === key })}
                  onClick={() => {
                    navigate(generatePath(tabs[key as keyof typeof tabs]?.path ?? '', { name }));
                  }}
                >
                  <FontAwesomeIcon icon={value.icon} className='mr-2' />
                  {value.label}
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
