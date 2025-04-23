import { useQuery } from '@connectrpc/connect-query';
import { faClockRotateLeft, faCog } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Result, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { Pipelines } from '@ui/features/project/pipelines-2/pipelines';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';
import { getProject } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Project as _Project } from '@ui/gen/api/v1alpha1/generated_pb';

export const Project = ({
  creatingStage,
  creatingWarehouse
}: {
  tab?: string;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name, stageName, promotionId, freight, stage } = useParams();

  const navigate = useNavigate();
  const projectBreadcrumbs = useProjectBreadcrumbs();

  const { data, isLoading, error } = useQuery(getProject, { name });

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

  return (
    <div className='h-full flex flex-col'>
      <BaseHeader>
        <Breadcrumb separator='>' items={[projectBreadcrumbs[0], { title: name }]} />
        <Space>
          <Button
            icon={<FontAwesomeIcon icon={faClockRotateLeft} size='sm' />}
            onClick={() => navigate(generatePath(paths.projectEvents, { name }))}
            size='small'
          >
            Events
          </Button>
          <Button
            icon={<FontAwesomeIcon icon={faCog} />}
            onClick={() => navigate(generatePath(paths.projectSettings, { name }))}
            size='small'
          >
            Settings
          </Button>
        </Space>
      </BaseHeader>

      <Pipelines
        project={data?.result?.value as _Project}
        stageName={stageName}
        promotionId={promotionId}
        promote={
          freight && stage
            ? {
                freight,
                stage
              }
            : undefined
        }
      />
    </div>
  );
};
