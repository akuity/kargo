import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';

import styles from './styles.module.less';

export const TruckIcon = (props: { className?: string }) => (
  <div className={classNames(props.className, styles.truckContainer)}>
    <FontAwesomeIcon icon={faTruckArrowRight} className={styles.truckIcon} />
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
  </div>
);
