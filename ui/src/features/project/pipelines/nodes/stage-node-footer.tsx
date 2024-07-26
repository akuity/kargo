import classNames from 'classnames';
import { formatDistance } from 'date-fns';

import * as styles from './stage-node.module.less';

export const StageNodeFooter = ({ lastPromotion }: { lastPromotion?: Date }) => {
  if (!lastPromotion) {
    return null;
  }

  return (
    <div className='flex flex-col w-full bg-gray-100 justify-center items-center p-1'>
      {lastPromotion && (
        <div className='flex items-center'>
          <div className={classNames(styles.smallLabel, '!mr-2')} style={{ paddingTop: '1px' }}>
            Last Promo:
          </div>
          <div className='text-xs text-gray-600 font-mono font-semibold'>
            {formatDistance(lastPromotion, new Date(), {
              addSuffix: true
            })}
          </div>
        </div>
      )}
    </div>
  );
};
