import { faClockRotateLeft, faCog } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { Pipelines } from '@ui/features/project/pipelines/pipelines';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

export const Project = ({
  creatingStage,
  creatingWarehouse
}: {
  tab?: string;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name } = useParams();

  const navigate = useNavigate();
  const projectBreadcrumbs = useProjectBreadcrumbs();

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
            icon={<FontAwesomeIcon icon={faCog} size='sm' />}
            onClick={() => navigate(generatePath(paths.projectSettings, { name }))}
            size='small'
          >
            Settings
          </Button>
        </Space>
      </BaseHeader>

      <Pipelines creatingStage={creatingStage} creatingWarehouse={creatingWarehouse} />
    </div>
  );
};
