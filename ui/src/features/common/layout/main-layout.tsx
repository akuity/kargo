import {
  faArrowRightFromBracket,
  faBook,
  faBoxes,
  faTerminal
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Spin, Tooltip } from 'antd';
import ErrorBoundary from 'antd/es/alert/ErrorBoundary';
import { Suspense } from 'react';
import { Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';

import * as styles from './main-layout.module.less';
import { NavItem } from './nav-item/nav-item';

export const MainLayout = () => {
  const { logout } = useAuthContext();

  return (
    <ErrorBoundary>
      <Suspense
        fallback={
          <div className='w-full h-screen flex items-center justify-center'>
            <Spin size='large' />
          </div>
        }
      >
        <div className={styles.wrapper}>
          <aside className={styles.sidebar}>
            <div className={styles.logo}>
              <img
                src='/kargo-icon.png'
                alt='Kargo Icon'
                className={styles.icon}
                width={50}
                height={31.5}
              />
              kargo
            </div>
            <Tooltip className={styles.version} title={__UI_VERSION__} placement='right'>
              {__UI_VERSION__ === 'development' ? 'dev' : __UI_VERSION__}
            </Tooltip>
            <nav className={styles.nav}>
              <NavItem icon={faBoxes} path={paths.projects}>
                Projects
              </NavItem>
              <NavItem icon={faBook} path='https://docs.kargo.io' target='_blank'>
                Docs
              </NavItem>
              <NavItem icon={faTerminal} path={paths.downloads}>
                CLI
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
      </Suspense>
    </ErrorBoundary>
  );
};
