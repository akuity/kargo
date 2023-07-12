import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { PropsWithChildren } from 'react';
import { NavLink } from 'react-router-dom';

import * as styles from './nav-item.module.less';

type Props = {
  path: string;
  icon: IconProp;
};

export const NavItem = ({ path, children, icon }: PropsWithChildren<Props>) => (
  <NavLink
    className={({ isActive }) => classNames(styles.navItem, { [styles.active]: isActive })}
    to={path}
  >
    <FontAwesomeIcon icon={icon} size='2x' />
    <span>{children}</span>
  </NavLink>
);
