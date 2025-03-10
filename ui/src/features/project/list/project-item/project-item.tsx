import classNames from 'classnames';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { StageTag } from '@ui/features/common/stage-tag';
import { getColors } from '@ui/features/stage/utils';
import { Project, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import * as styles from './project-item.module.less';

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
            <StageTag
              key={stage.metadata?.name}
              stage={stage}
              projectName={project?.metadata?.name || ''}
              stageColorMap={stageColorMap}
            />
          ))}
        </div>
      )}
    </Link>
  );
};
