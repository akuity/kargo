import { useQuery } from '@tanstack/react-query';

import { listProjects } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

export const ProjectsList = () => {
  const { data } = useQuery(listProjects.useQuery({}));

  return (
    <>
      <div className={styles.list}>
        {data?.projects.map((project) => (
          <ProjectItem key={project.name} name={project.name} />
        ))}
      </div>
    </>
  );
};
