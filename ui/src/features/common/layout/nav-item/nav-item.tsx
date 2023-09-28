import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { PropsWithChildren } from 'react';
import { NavLink } from 'react-router-dom';

import * as styles from './nav-item.module.less';

type Props = {
  path: string;
  icon: IconProp;
  target?: string;
};

export const NavItem = ({ path, children, icon, target }: PropsWithChildren<Props>) => (
  <NavLink
    className={({ isActive }) => classNames(styles.navItem, { [styles.active]: isActive })}
    to={path}
    target={target}
  >
    <FontAwesomeIcon icon={icon} size='2x' />
    <span>{children}</span>
  </NavLink>
);
