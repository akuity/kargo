import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import styles from './styles.module.less';

export const TruckIcon = () => (
  <div className={styles.truckContainer}>
    <FontAwesomeIcon icon={faTruckArrowRight} className={styles.truckIcon} />
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
    <div className={styles.exhaustParticle}></div>
  </div>
);
