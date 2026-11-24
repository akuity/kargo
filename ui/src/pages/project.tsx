import { faBullseye, faClockRotateLeft, faCog } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { Pipelines } from '@ui/features/project/pipelines/pipelines';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';
import { useListTargets } from '@ui/gen/api/v2/core/core';

export const Project = ({
  creatingStage,
  creatingWarehouse
}: {
  tab?: string;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name, stageName, warehouseName, promotionId, freightName } = useParams();
  const navigate = useNavigate();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  const { projectSubpages } = useExtensionsContext();

  // The Targets page only means something once the project has Targets, so
  // its button appears only then.
  const targetsQuery = useListTargets(name || '', undefined);
  const showTargets = (targetsQuery.data?.data?.items?.length || 0) > 0;

  const resourceLabel =
    (stageName && `Stage: ${stageName}`) ||
    (warehouseName && `Warehouse: ${warehouseName}`) ||
    (promotionId && `Promotion: ${promotionId}`) ||
    (freightName && `Freight: ${freightName}`);
  const titleParts = resourceLabel ? [resourceLabel, name] : [name];
  useDocumentTitle(titleParts);

  return (
    <div className='h-full flex flex-col'>
      <BaseHeader>
        <Breadcrumb separator='>' items={[projectBreadcrumbs[0], { title: name }]} />
        <Space>
          {projectSubpages.map((page) => (
            <Button
              key={page.path}
              icon={page.icon ? <FontAwesomeIcon icon={page.icon} size='sm' /> : null}
              onClick={() =>
                navigate(`${generatePath(paths.projectExtensions, { name })}/${page.path}`)
              }
              size='small'
            >
              {page.label}
            </Button>
          ))}
          {showTargets && (
            <Button
              icon={<FontAwesomeIcon icon={faBullseye} size='sm' />}
              onClick={() => navigate(generatePath(paths.projectTargets, { name }))}
              size='small'
            >
              Targets
            </Button>
          )}
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
