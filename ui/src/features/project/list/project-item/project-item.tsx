import { Link } from 'react-router-dom';

import * as styles from './project-item.module.less';

type Props = {
  name: string;
};

export const ProjectItem = ({ name }: Props) => (
  <Link className={styles.tile} to={`/project/${name}`}>
    <div className={styles.title}>{name}</div>
  </Link>
);
