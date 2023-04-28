import { paths } from '@config/paths';
import { faTableList } from '@fortawesome/free-solid-svg-icons';
import { Outlet } from 'react-router-dom';

import * as styles from './main-layout.module.less';
import { NavItem } from './nav-item/nav-item';

export const MainLayout = () => (
  <div className={styles.wrapper}>
    <aside className={styles.sidebar}>
      <div className={styles.logo}>Kargo</div>
      <nav>
        <NavItem icon={faTableList} path={paths.projects}>
          Namespaces
        </NavItem>
      </nav>
    </aside>
    <div className={styles.contentWrapper}>
      <div className={styles.content}>
        <Outlet />
      </div>
    </div>
  </div>
);
