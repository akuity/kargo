import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { Ref } from 'react';

import styles from './styles.module.less';

// AntD v6 positions popups from a ref on the trigger -- v5 could fall back to
// the DOM node. A child that drops `ref` leaves Tooltip with nothing to anchor
// to, so it never opens.
export const TruckIcon = ({
  className,
  ref
}: {
  className?: string;
  ref?: Ref<HTMLDivElement>;
}) => (
  <div ref={ref} className={classNames(className, styles.truckContainer)}>
    <FontAwesomeIcon icon={faTruckArrowRight} className={styles.truckIcon} />
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
  </div>
);
