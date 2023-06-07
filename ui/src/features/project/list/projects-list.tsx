import { Input } from 'antd';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

const testProjects = ['kargo-demo'];

export const ProjectsList = () => {
  return (
    <>
      <Input placeholder='Search...' size='large' />
      <div className={styles.list}>
        {testProjects.map((project) => (
          <ProjectItem key={project} name={project} />
        ))}
      </div>
    </>
  );
};
