import classNames from 'classnames';

import { Freight } from '@ui/gen/v1alpha1/types_pb';

import { FreightLabel } from '../common/freight-label';

import styles from './freightline.module.less';

export enum FreightMode {
  Default = 'default', // not promoting, has stages
  Promotable = 'promotable', // promoting, promotable
  Disabled = 'disabled',
  Selected = 'selected',
  Confirming = 'confirming' // promoting, confirming
}

export const FreightItem = ({
  freight,
  children,
  onClick,
  mode,
  empty
}: {
  freight?: Freight;
  children: React.ReactNode;
  onClick?: () => void;
  mode: FreightMode;
  empty: boolean;
}) => {
  return (
    <div
      className={classNames('relative h-full', styles.freightItem, {
        ['w-32']: !empty && mode !== FreightMode.Confirming,
        ['border-gray-500']: mode === FreightMode.Default && !empty,
        [styles.promotable]: mode === FreightMode.Promotable,
        [styles.disabled]: mode === FreightMode.Disabled,
        [styles.confirming]: mode === FreightMode.Confirming,
        [styles.selected]: mode === FreightMode.Selected
      })}
      onClick={onClick}
    >
      <div className='flex w-full h-full mb-1 items-center justify-center'>{children}</div>
      <div className='mt-auto w-full'>
        <div
          className={`w-full text-center font-mono text-xs truncate ${
            mode === FreightMode.Confirming ? 'text-white' : 'text-gray-400'
          }`}
        >
          <FreightLabel freight={freight} />
        </div>
      </div>
    </div>
  );
};
