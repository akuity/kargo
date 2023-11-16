import { faArrowRightFromBracket, faBook, faBoxes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button } from 'antd';
import { Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';

import * as styles from './main-layout.module.less';
import { NavItem } from './nav-item/nav-item';

export const MainLayout = () => {
  const { logout } = useAuthContext();

  return (
    <div className={styles.wrapper}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>
          <img src='/kargo-icon.png' alt='Kargo Icon' className={styles.icon} />
          kargo
        </div>
        <nav className={styles.nav}>
          <NavItem icon={faBoxes} path={paths.projects}>
            Projects
          </NavItem>
          <NavItem icon={faBook} path='https://kargo.akuity.io' target='_blank'>
            Docs
          </NavItem>
        </nav>

        <Button
          className={styles.logout}
          onClick={logout}
          type='text'
          icon={<FontAwesomeIcon icon={faArrowRightFromBracket} />}
        >
          Logout
        </Button>
      </aside>
      <div className={styles.contentWrapper}>
        <div className={styles.content}>
          <Outlet />
        </div>
      </div>
    </div>
  );
};
