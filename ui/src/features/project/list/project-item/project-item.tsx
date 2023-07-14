import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';

import * as styles from './project-item.module.less';

type Props = {
  name: string;
};

export const ProjectItem = ({ name }: Props) => (
  <Link className={styles.tile} to={generatePath(paths.project, { name })}>
    <div className={styles.title}>{name}</div>
  </Link>
);
