import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { PromotionType } from './freightline';

export const PromotingStageBanner = ({ promotionType }: { promotionType: PromotionType }) => {
  return (
    <>
      <FontAwesomeIcon icon={faTriangleExclamation} className='mr-2' />
      Available freight includes all which have been verified in{' '}
      {promotionType === 'subscribers' ? (
        <>this stage.</>
      ) : (
        <> any immediately upstream stage OR approved for this stage.</>
      )}
    </>
  );
};
