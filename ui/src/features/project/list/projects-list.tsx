import { useQuery } from '@connectrpc/connect-query';
import { Empty } from 'antd';

import { LoadingState } from '@ui/features/common';
import { listDetailedProjects } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

export const ProjectsList = () => {
  const { data, isLoading } = useQuery(listDetailedProjects, {});

  if (isLoading) return <LoadingState />;

  if (!data || data.detailedProjects.length === 0) return <Empty />;

  return (
    <div className={styles.list}>
      {data.detailedProjects.map((proj) => (
        <ProjectItem key={proj.project?.metadata?.name} project={proj} />
      ))}
    </div>
  );
};
