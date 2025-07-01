import {
  faArrowRightFromBracket,
  faGear,
  faBook,
  faBoxes,
  faTerminal,
  faUser
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Spin, Tooltip } from 'antd';
import ErrorBoundary from 'antd/es/alert/ErrorBoundary';
import { Suspense } from 'react';
import { Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { isJWTDirty } from '@ui/features/auth/jwt-utils';
import { KargoLogo } from '@ui/features/common/logo/logo';

import * as styles from './main-layout.module.less';
import { NavItem } from './nav-item/nav-item';

export const MainLayout = () => {
  const { logout, JWTInfo } = useAuthContext();
  const { appSubpages, layoutExtensions } = useExtensionsContext();

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
            <KargoLogo className='my-4' />
            <Tooltip className={styles.version} title={__UI_VERSION__} placement='right'>
              {__UI_VERSION__ === 'development' ? 'dev' : __UI_VERSION__}
            </Tooltip>
            <nav className={styles.nav}>
              <NavItem icon={faBoxes} path={paths.projects}>
                Projects
              </NavItem>
              {!isJWTDirty(JWTInfo) && (
                <NavItem icon={faUser} path={paths.user}>
                  User
                </NavItem>
              )}
              {appSubpages.map((page) => (
                <NavItem
                  key={page.path}
                  icon={page.icon}
                  path={`${paths.appExtensions}/${page.path}`}
                >
                  {page.label}
                </NavItem>
              ))}
              <NavItem icon={faGear} path={paths.settings}>
                Settings
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
        {layoutExtensions.map(({ component: Comp }, index) => (
          <Comp key={index} />
        ))}
      </Suspense>
    </ErrorBoundary>
  );
};
