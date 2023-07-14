import { FontAwesomeIcon, FontAwesomeIconProps } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';

import * as styles from './button-icon.module.less';

interface Props extends FontAwesomeIconProps {
  noMargin?: boolean;
}

export const ButtonIcon = ({ noMargin = false, ...props }: Props) => (
  <FontAwesomeIcon
    {...props}
    className={classNames({ [styles.withMargin]: !noMargin }, props.className)}
  />
);
