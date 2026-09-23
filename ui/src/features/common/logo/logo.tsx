import classNames from 'classnames';

import { withBasePath } from '@ui/config/base-path';

import styles from './logo.module.less';
import { KargoLogoVariants } from './types';

type KargoLogoProps = {
  className?: string;
  variant?: KargoLogoVariants;
};

const getLogo = (variant?: KargoLogoVariants) => {
  switch (variant) {
    case KargoLogoVariants.LIGHT_BACKGROUND:
      return withBasePath('/kargo-logo.png');
    case KargoLogoVariants.HEAD_ONLY:
      return withBasePath('/kargo-logo-head-only.png');
  }
  return withBasePath('/kargo-logo-white.png');
};

export const KargoLogo = (props: KargoLogoProps) => (
  <div className={classNames(props.className, styles.logo)}>
    <img src={getLogo(props.variant)} alt='Kargo Logo' />
  </div>
);
