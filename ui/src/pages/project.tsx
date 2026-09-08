import { faClockRotateLeft, faCog } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { Fleet } from '@ui/features/project/fleet/fleet';
import { Pipelines } from '@ui/features/project/pipelines/pipelines';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

export const Project = ({
  tab,
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

  const resourceLabel =
    (stageName && `Stage: ${stageName}`) ||
    (warehouseName && `Warehouse: ${warehouseName}`) ||
    (promotionId && `Promotion: ${promotionId}`) ||
    (freightName && `Freight: ${freightName}`) ||
    (tab === 'fleet' && 'Fleet');
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

      {/* The Fleet view has no entry point in the header yet; it is reached by
          navigating to /project/:name/fleet directly. */}
      {tab === 'fleet' ? (
        <Fleet />
      ) : (
        <Pipelines creatingStage={creatingStage} creatingWarehouse={creatingWarehouse} />
      )}
    </div>
  );
};
