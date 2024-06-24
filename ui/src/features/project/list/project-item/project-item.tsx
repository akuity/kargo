import { Tooltip } from 'antd';
import classNames from 'classnames';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getStageColors } from '@ui/features/stage/utils';
import { DetailedProject } from '@ui/gen/service/v1alpha1/service_pb';
import { Project } from '@ui/gen/v1alpha1/generated_pb';

import * as styles from './project-item.module.less';
import { StagePopover } from './stage-popover';

export const ProjectItem = ({ project }: { project?: DetailedProject }) => {
  const stageColorMap = getStageColors(
    project?.project?.metadata?.name || '',
    project?.stages || []
  );

  return (
    <Link
      className={styles.tile}
      to={generatePath(paths.project, { name: project?.project?.metadata?.name })}
    >
      <div className={classNames(styles.title, 'mb-2')}>{project?.project?.metadata?.name}</div>
      <Description item={project?.project as Project} loading={false} className='mb-4' />
      <div className='flex items-center gap-x-3 gap-y-1 flex-wrap'>
        {project?.stages?.map((stage) => (
          <Tooltip
            key={stage.metadata?.name}
            placement='bottom'
            title={
              stage?.status?.lastPromotion?.name && (
                <StagePopover
                  promotionName={stage?.status?.lastPromotion?.name}
                  project={project?.project?.metadata?.name}
                  freightName={stage?.status?.currentFreight?.name}
                  stageName={stage?.metadata?.name}
                />
              )
            }
          >
            <div
              className='flex items-center mb-2 text-white rounded py-1 px-2 font-semibold bg-gray-600'
              style={{ backgroundColor: stageColorMap[stage.metadata?.name || ''] }}
            >
              {stage.status?.health && (
                <div className='mr-2'>
                  <HealthStatusIcon health={stage.status?.health} hideColor={true} />
                </div>
              )}
              {!stage?.status?.currentPromotion && stage.status?.lastPromotion && (
                <div className='mr-2'>
                  <PromotionStatusIcon
                    placement='top'
                    status={stage.status?.lastPromotion?.status}
                    color='white'
                    size='1x'
                  />
                </div>
              )}
              {stage.metadata?.name}
            </div>
          </Tooltip>
        ))}
      </div>
    </Link>
  );
};
