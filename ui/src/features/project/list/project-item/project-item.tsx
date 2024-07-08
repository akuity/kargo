import { Tooltip } from 'antd';
import classNames from 'classnames';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getColors } from '@ui/features/stage/utils';
import { Project, Stage } from '@ui/gen/v1alpha1/generated_pb';

import * as styles from './project-item.module.less';
import { StagePopover } from './stage-popover';

export const ProjectItem = ({ project, stages }: { project?: Project; stages?: Stage[] }) => {
  const stageColorMap = getColors(project?.metadata?.name || '', stages || []);

  return (
    <Link
      className={styles.tile}
      to={generatePath(paths.project, { name: project?.metadata?.name })}
    >
      <div className={classNames(styles.title, 'mb-2')}>{project?.metadata?.name}</div>
      <Description item={project as Project} loading={false} />
      {(stages || []).length > 0 && (
        <div className='flex items-center gap-x-3 gap-y-1 flex-wrap mt-4'>
          {stages?.map((stage) => (
            <Tooltip
              key={stage.metadata?.name}
              placement='bottom'
              title={
                stage?.status?.lastPromotion?.name && (
                  <StagePopover project={project?.metadata?.name} stage={stage} />
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
      )}
    </Link>
  );
};
