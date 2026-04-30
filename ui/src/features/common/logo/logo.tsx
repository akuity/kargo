import classNames from 'classnames';

import styles from './logo.module.less';
import { KargoLogoVariants } from './types';

type KargoLogoProps = {
  className?: string;
  variant?: KargoLogoVariants;
};

const getLogo = (variant?: KargoLogoVariants) => {
  switch (variant) {
    case KargoLogoVariants.LIGHT_BACKGROUND:
      return '/kargo-logo.png';
    case KargoLogoVariants.HEAD_ONLY:
      return '/kargo-logo-head-only.png';
  }
  return '/kargo-logo-white.png';
};

export const KargoLogo = (props: KargoLogoProps) => (
  <div className={classNames(props.className, styles.logo)}>
    <img src={getLogo(props.variant)} alt='Kargo Logo' />
  </div>
);
