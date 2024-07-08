import classNames from 'classnames';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';

import { FreightLabel } from '../common/freight-label';
import { FreightMode } from '../project/pipelines/types';

import styles from './freight-timeline.module.less';

export const FreightItem = ({
  freight,
  children,
  onClick,
  mode,
  empty,
  highlighted,
  onHover,
  hideLabel
}: {
  freight?: Freight;
  children: React.ReactNode;
  onClick?: () => void;
  mode: FreightMode;
  empty: boolean;
  highlighted?: boolean;
  onHover: (hovering: boolean) => void;
  hideLabel?: boolean;
}) => {
  return (
    <div
      className={classNames('relative h-full cursor-pointer', styles.freightItem, {
        ['w-32']: !empty && mode !== FreightMode.Confirming,
        ['w-24']: empty || mode === FreightMode.Confirming,
        [styles.notEmpty]: mode === FreightMode.Default && !empty,
        [styles.promotable]: mode === FreightMode.Promotable,
        [styles.disabled]: mode === FreightMode.Disabled,
        [styles.confirming]: mode === FreightMode.Confirming,
        [styles.selected]: mode === FreightMode.Selected,
        [styles.highlighted]: highlighted
      })}
      onClick={onClick}
      onMouseEnter={() => onHover(true)}
      onMouseLeave={() => onHover(false)}
    >
      <div className='flex w-full h-full mb-1 items-center justify-center max-w-full text-ellipsis overflow-hidden'>
        {children}
      </div>
      <div className='mt-auto w-full'>
        <div
          className={`w-full text-center font-mono text-xs truncate ${
            mode === FreightMode.Confirming ? 'text-black' : 'text-gray-400'
          }`}
        >
          {!hideLabel && <FreightLabel freight={freight} breakOnHyphen={true} />}
        </div>
      </div>
    </div>
  );
};
