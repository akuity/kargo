import { useQuery } from '@tanstack/react-query';
import { Empty } from 'antd';

import { LoadingState } from '@ui/features/common';
import { listProjects } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

export const ProjectsList = () => {
  const { data, isLoading } = useQuery(listProjects.useQuery({}));

  if (isLoading) return <LoadingState />;

  if (!data || data.projects.length === 0) return <Empty />;

  return (
    <>
      <div className={styles.list}>
        {data.projects.map((project) => (
          <ProjectItem key={project.name} name={project.name} />
        ))}
      </div>
    </>
  );
};
