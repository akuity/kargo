import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { PropsWithChildren } from 'react';
import { matchPath, NavLink, useLocation } from 'react-router-dom';

type Variant = 'primary' | 'resource';

type Props = {
  path: string;
  icon: IconProp;
  // 'primary' is main navigation; 'resource' is the smaller, secondary style
  // used in the sidebar's Resources tray.
  variant?: Variant;
  // Opens `path` in a new tab via a plain anchor. Never shown as the current page.
  external?: boolean;
  trailingIcon?: IconProp;
  // Extra route patterns that also mark this item as current, including
  // everything beneath them (e.g. '/project/:name' covers its stages).
  activePaths?: string[];
};

// The sidebar is dark in both themes, so it reuses AntD's dark-theme values:
// text white/85 (colorText) and /65 (colorTextSecondary), fills white/[.08]
// (colorFillTertiary) and /[.12] (colorFillSecondary), and the dark theme's
// colorLink (#7aa0d6, config/themeConfig.ts) as the accent.
// Items fill their nav's width; the sidebar sets the gutters.
const itemBase =
  'relative flex w-full flex-col items-center justify-center no-underline transition-colors duration-100 ' +
  'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#7aa0d6]';

// AntD's global link style turns every `a:hover` / `a:active` blue and outranks
// a single class, so each state pins its own hover and active text colour.
const variants: Record<
  Variant,
  { item: string; idle: string; current: string; icon: string; currentIcon: string }
> = {
  primary: {
    item: 'h-[60px] gap-1.5 rounded-[10px] text-[11px]',
    idle: 'text-white/85 hover:text-white active:text-white',
    // Accent-tinted, so the current page reads as selected, not just hovered.
    current: 'bg-[#7aa0d6]/10 text-white hover:text-white active:text-white',
    icon: 'text-[20px]',
    currentIcon: 'text-[#7aa0d6]'
  },
  resource: {
    item: 'h-12 gap-1 rounded-lg text-[10px]',
    idle: 'text-white/65 hover:bg-white/[.08] hover:text-white active:text-white',
    // Grey fill only -- no accent colour for in-app utilities.
    current: 'bg-white/[.12] text-white/85 hover:text-white/85 active:text-white/85',
    icon: 'text-[16px]',
    currentIcon: ''
  }
};

export const NavItem = ({
  path,
  icon,
  variant = 'primary',
  external = false,
  trailingIcon,
  activePaths,
  children
}: PropsWithChildren<Props>) => {
  const { pathname } = useLocation();
  const inActivePaths = Boolean(
    activePaths?.some((pattern) => matchPath({ path: pattern, end: false }, pathname) !== null)
  );
  const v = variants[variant];

  const itemClass = (current: boolean) =>
    classNames(itemBase, v.item, current ? v.current : v.idle);

  const renderContent = (current: boolean) => (
    <>
      <FontAwesomeIcon
        icon={icon}
        className={classNames('shrink-0', v.icon, current && v.currentIcon)}
      />
      {/* Long extension labels truncate instead of widening the rail. */}
      <span className='flex max-w-full items-center gap-[3px] px-1 leading-[1.2]'>
        <span className='min-w-0 truncate'>{children}</span>
        {trailingIcon && (
          <FontAwesomeIcon icon={trailingIcon} className='shrink-0 text-[8px] opacity-70' />
        )}
      </span>
    </>
  );

  return external ? (
    <a className={itemClass(false)} href={path} target='_blank' rel='noopener noreferrer'>
      {renderContent(false)}
      <span className='sr-only'> (opens in a new tab)</span>
    </a>
  ) : (
    <NavLink to={path} className={({ isActive }) => itemClass(isActive || inActivePaths)}>
      {({ isActive }) => renderContent(isActive || inActivePaths)}
    </NavLink>
  );
};
