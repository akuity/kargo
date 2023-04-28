import * as styles from './project-item.module.less';

type Props = {
  namespace: string;
};

export const ProjectItem = ({ namespace }: Props) => <div className={styles.tile}>{namespace}</div>;
