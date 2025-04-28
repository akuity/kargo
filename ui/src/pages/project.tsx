import { useQuery } from '@connectrpc/connect-query';
import { faClockRotateLeft, faCog, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Dropdown, Result, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { Pipelines } from '@ui/features/project/pipelines-2/pipelines';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';
import {
  getProject,
  listWarehouses
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Project as _Project } from '@ui/gen/api/v1alpha1/generated_pb';

export const Project = ({
  creatingStage,
  creatingWarehouse
}: {
  tab?: string;
  creatingStage?: boolean;
  creatingWarehouse?: boolean;
}) => {
  const { name, stageName, promotionId, freight, stage, warehouseName } = useParams();

  const navigate = useNavigate();
  const projectBreadcrumbs = useProjectBreadcrumbs();

  const { data, isLoading, error } = useQuery(getProject, { name });

  const listWarehousesQuery = useQuery(
    listWarehouses,
    {
      project: name
    },
    { enabled: !isLoading }
  );

  if (isLoading || listWarehousesQuery.isLoading) {
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
            size='small'
            icon={<FontAwesomeIcon icon={faPlus} />}
            onClick={() => navigate(generatePath(paths.createStage, { name }))}
          >
            Create Stage
          </Button>
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faPlus} />}
            onClick={() => navigate(generatePath(paths.createWarehouse, { name }))}
          >
            Create Warehouse
          </Button>
          <Dropdown
            menu={{
              items: listWarehousesQuery.data?.warehouses?.map((warehouse) => ({
                key: warehouse?.metadata?.name || '',
                label: warehouse?.metadata?.name || '',
                onClick: () => {
                  navigate(
                    generatePath(paths.warehouse, {
                      name,
                      warehouseName: warehouse?.metadata?.name || '',
                      tab: 'create-freight'
                    })
                  );
                }
              }))
            }}
            trigger={['click']}
          >
            <Button size='small' icon={<FontAwesomeIcon icon={faPlus} />}>
              Create Freight
            </Button>
          </Dropdown>
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
        creatingStage={creatingStage}
        creatingWarehouse={creatingWarehouse}
        warehouses={listWarehousesQuery.data?.warehouses || []}
        warehouseName={warehouseName}
      />
    </div>
  );
};
