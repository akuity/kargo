import { faQuestionCircle, faTriangleExclamation } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { useLocalStorage } from '@ui/utils/use-local-storage';

import { PromotionType } from './freightline';

export const PromotingStageBanner = ({ promotionType }: { promotionType: PromotionType }) => {
  const [dismissed, setDismissed] = useLocalStorage('promoting-stage-banner-dismissed');
  return (
    <div className='flex items-center w-full'>
      {dismissed === 'true' ? (
        <>
          <FontAwesomeIcon
            icon={faQuestionCircle}
            onClick={() => setDismissed('false')}
            className='cursor-pointer'
          />
        </>
      ) : (
        <>
          <FontAwesomeIcon icon={faTriangleExclamation} className='mr-2' />
          Available freight includes all which have been verified in{' '}
          {promotionType === 'subscribers' ? (
            <>this stage.</>
          ) : (
            <> any immediately upstream stage OR approved for this stage.</>
          )}
          <div
            className='ml-auto mr-2 cursor-pointer rounded bg-gray-600 px-2'
            onClick={() => setDismissed('true')}
          >
            DISMISS
          </div>
        </>
      )}
    </div>
  );
};
