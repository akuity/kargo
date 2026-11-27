import {
  faArrowRightFromBracket,
  faArrowUpRightFromSquare,
  faBook,
  faBoxes,
  faGear,
  faTerminal,
  faUser
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  ConfigProvider,
  Dropdown,
  Layout,
  MenuProps,
  Spin,
  ThemeConfig,
  Tooltip,
  Typography
} from 'antd';
import ErrorBoundary from 'antd/es/alert/ErrorBoundary';
import classNames from 'classnames';
import { Suspense, useState } from 'react';
import { Link, Outlet, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { themeConfig } from '@ui/config/themeConfig';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { asText, getDisplayName, getUserEmail, isJWTDirty } from '@ui/features/auth/jwt-utils';
import { KargoLogo } from '@ui/features/common/logo/logo';

import { NavItem } from './nav-item/nav-item';

// Keep in sync with the rail's `w-[100px]` and the avatar's `size-9` below.
const SIDEBAR_WIDTH = 100;
const AVATAR_SIZE = 36;
const ACCOUNT_MENU_GAP = 8;

// The avatar is centred in the rail, so shift the menu by the avatar's
// distance to the rail's edge to open it beside the sidebar, not over it.
const accountMenuAlign = {
  offset: [(SIDEBAR_WIDTH - AVATAR_SIZE) / 2 + ACCOUNT_MENU_GAP, 0]
};

// The account menu opens off the rail, so it stays dark in both themes too.
// `inherit: false` keeps the light app theme from leaking in.
const accountMenuTheme: ThemeConfig = { ...themeConfig(true), inherit: false };

// `!` outranks AntD's own dropdown menu styles.
const accountMenuClass =
  '!min-w-[236px] !rounded-[10px] !border !border-solid !border-white/[.08] !p-1.5';

// Rail colours reuse existing values -- AntD's dark-theme alphas and
// config/themeConfig.ts colours; see the note in nav-item/nav-item.tsx.
// Avatar: brand navy (light colorPrimary) with colorText-level white.
const avatarClass =
  'flex size-9 cursor-pointer items-center justify-center rounded-full border-0 bg-[#30476c] p-0 ' +
  'text-sm font-semibold leading-none text-white/85 transition-shadow duration-150 ' +
  'hover:shadow-[0_0_0_2px_#7aa0d6] ' +
  // Offset past the hover/open ring so focus stays distinguishable.
  'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[3px] focus-visible:outline-[#7aa0d6]';

// Separators inside the rail.
const railLine = 'border-0 border-solid border-white/[.08]';

const AccountMenu = () => {
  const { logout, JWTInfo } = useAuthContext();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  const displayName = getDisplayName(JWTInfo);
  const email = asText(getUserEmail(JWTInfo));
  // The name can fall back to the email, and the local admin's "email" is
  // 'Admin', so skip the second line when it would repeat the name.
  const showEmail = email.toLowerCase() !== displayName.toLowerCase();

  const items: MenuProps['items'] = [
    {
      key: 'header',
      type: 'group',
      label: (
        // Group titles take the theme's secondary text colour; the name opts
        // back into the primary one.
        <div className='max-w-[240px] px-1 py-1.5'>
          <div className='truncate'>
            <Typography.Text strong>{displayName}</Typography.Text>
          </div>
          {showEmail && <div className='truncate text-xs'>{email || 'Signed in'}</div>}
        </div>
      )
    },
    { type: 'divider' },
    isJWTDirty(JWTInfo)
      ? null
      : { key: 'user', label: 'Your account', icon: <FontAwesomeIcon icon={faUser} /> },
    {
      key: 'logout',
      // Red text rather than `danger: true`, which fills the row red on hover.
      label: <Typography.Text type='danger'>Log out</Typography.Text>,
      icon: (
        <Typography.Text type='danger'>
          <FontAwesomeIcon icon={faArrowRightFromBracket} />
        </Typography.Text>
      )
    }
  ];

  const onMenuClick: MenuProps['onClick'] = ({ key }) => {
    setOpen(false);
    if (key === 'user') {
      navigate(paths.user);
    } else if (key === 'logout') {
      logout();
    }
  };

  return (
    <ConfigProvider theme={accountMenuTheme}>
      <Dropdown
        open={open}
        onOpenChange={setOpen}
        // 'click' covers touch and keyboard (Enter/Space) alongside hover. With
        // 'hover' alone, rc-trigger handles taps itself and then ignores the
        // mouse for the rest of the session on touchscreen laptops.
        trigger={['hover', 'click']}
        // Hover opens the menu before a click lands, so a click must only open
        // it, never toggle it shut. It still closes on leaving, Escape or a click
        // outside. AntD forwards hideAction to rc-trigger but doesn't declare it,
        // hence the spread.
        {...{ hideAction: ['hover'] }}
        // Time to cross the gap between the avatar and the menu.
        mouseLeaveDelay={0.3}
        placement='rightBottom'
        align={accountMenuAlign}
        menu={{ items, onClick: onMenuClick, className: accountMenuClass }}
      >
        <button
          type='button'
          className={classNames(avatarClass, { 'shadow-[0_0_0_2px_#7aa0d6]': open })}
          aria-label={`Account: ${displayName}`}
          aria-haspopup='menu'
          aria-expanded={open}
        >
          {Array.from(displayName)[0].toUpperCase()}
        </button>
      </Dropdown>
    </ConfigProvider>
  );
};

export const MainLayout = () => {
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
        <Layout>
          <div className='flex max-h-screen min-h-screen'>
            {/* Dark in both themes: #111 in light; chrome tone + divider in dark so it
                doesn't blend into the page. Scrolls itself when the window is too short. */}
            <aside
              className={classNames(
                'flex w-[100px] flex-none flex-col items-center overflow-y-auto overflow-x-hidden text-white',
                'bg-[color:var(--kargo-sidebar-bg,#111)]',
                'border-0 border-r border-solid border-[color:var(--kargo-sidebar-border,transparent)]',
                '[scrollbar-color:rgba(255,255,255,0.18)_transparent] [scrollbar-width:thin]'
              )}
            >
              <div
                className={classNames(
                  'flex w-full shrink-0 justify-center border-b px-2 py-4',
                  railLine
                )}
              >
                <Link
                  to={paths.home}
                  className='block rounded-lg focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#7aa0d6]'
                  title='Home'
                >
                  {/* `!` outranks KargoLogo's default 64px width. */}
                  <KargoLogo className='[&_img]:!w-[52px]' />
                </Link>
              </div>

              <nav
                aria-label='Main'
                className='mt-3.5 flex shrink-0 flex-col items-center gap-1 self-stretch px-2'
              >
                <NavItem
                  icon={faBoxes}
                  path={paths.projects}
                  activePaths={[paths.project, paths.createProjectGuided]}
                >
                  Projects
                </NavItem>
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
              </nav>

              <div className='min-h-4 flex-auto' />

              <nav
                aria-label='Resources'
                className='mx-2 mb-2.5 flex shrink-0 flex-col items-center gap-0.5 self-stretch rounded-xl border border-solid border-white/[.08] bg-white/[.04] p-1'
              >
                <NavItem variant='resource' icon={faTerminal} path={paths.downloads}>
                  CLI
                </NavItem>
                <NavItem
                  variant='resource'
                  external
                  icon={faBook}
                  path='https://docs.kargo.io'
                  trailingIcon={faArrowUpRightFromSquare}
                >
                  Docs
                </NavItem>
              </nav>

              <div className='mb-2.5 flex shrink-0 justify-center self-stretch px-2'>
                <Tooltip title={__UI_VERSION__} placement='right'>
                  <div className='max-w-full truncate font-mono text-[10px] leading-3 text-white/65'>
                    {__UI_VERSION__ === 'development' ? 'dev' : __UI_VERSION__}
                  </div>
                </Tooltip>
              </div>

              <div
                className={classNames(
                  'flex shrink-0 justify-center self-stretch border-t py-3.5',
                  railLine
                )}
              >
                <AccountMenu />
              </div>
            </aside>
            <div className='relative flex-1 overflow-hidden bg-[color:var(--kargo-color-bg-layout,#f7f8fa)]'>
              <div className='relative h-full w-full overflow-auto'>
                <Outlet />
              </div>
            </div>
          </div>
          {layoutExtensions.map(({ component: Comp }, index) => (
            <Comp key={index} />
          ))}
        </Layout>
      </Suspense>
    </ErrorBoundary>
  );
};
