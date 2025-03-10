import classNames from 'classnames';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightMode } from '../project/pipelines/types';

import { FreightItemLabel } from './freight-item-label';
import styles from './freight-timeline.module.less';

export const FreightItem = ({
  freight,
  children,
  onClick,
  mode,
  empty,
  highlighted,
  onHover,
  hideLabel,
  childClassname
}: {
  freight?: Freight;
  children: React.ReactNode;
  onClick?: () => void;
  mode: FreightMode;
  empty: boolean;
  highlighted?: boolean;
  onHover: (hovering: boolean) => void;
  hideLabel?: boolean;
  childClassname?: string;
}) => {
  let width = '';
  if (mode !== FreightMode.Confirming) {
    if (empty) {
      width = '96px';
    } else {
      width = '135px';
    }
  }
  return (
    <div
      className={classNames('relative h-full cursor-pointer', styles.freightItem, {
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
      style={{
        width
      }}
    >
      <div
        className={classNames(
          'flex w-full h-full mb-1 items-center justify-center max-w-full text-ellipsis overflow-hidden',
          childClassname
        )}
      >
        {children}
      </div>
      <div className='mt-auto w-full'>
        <div
          className={`w-full text-center font-mono text-xs truncate ${
            mode === FreightMode.Confirming ? 'text-black' : 'text-gray-400'
          }`}
        >
          {!hideLabel && <FreightItemLabel freight={freight} />}
        </div>
      </div>
    </div>
  );
};
