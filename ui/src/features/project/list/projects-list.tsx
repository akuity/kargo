import { Input } from 'antd';

import { ProjectItem } from './project-item/project-item';
import * as styles from './projects-list.module.less';

export const ProjectsList = () => {
  return (
    <>
      <Input placeholder='Search...' size='large' />
      <div className={styles.list}>
        {['kargo-demo', 'simple-demo', 'hello-world'].map((item) => (
          <ProjectItem key={item} namespace={item} />
        ))}
      </div>
    </>
  );
};
