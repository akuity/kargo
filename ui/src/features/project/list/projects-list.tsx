import { transport } from '@config/transport';
import { listProjects } from '@gen/service/v1alpha1/service-KargoService_connectquery';
import { useQuery } from '@tanstack/react-query';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

export const ProjectsList = () => {
  const { data } = useQuery(listProjects.useQuery({}, { transport }));

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
