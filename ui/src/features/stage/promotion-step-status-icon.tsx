import {
  faBan,
  faCheck,
  faCircleNotch,
  faRotate,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';

export const StepStatusIcon = ({ status }: { status: PromotionDirectiveStepStatus }) => (
  <Flex
    align='center'
    justify='center'
    className='mr-2'
    style={{ width: '20px', height: '20px', marginBottom: '1px' }}
  >
    {status === PromotionDirectiveStepStatus.RUNNING && (
      <FontAwesomeIcon spin icon={faCircleNotch} />
    )}
    {status === PromotionDirectiveStepStatus.RETRYING && (
      <FontAwesomeIcon spin icon={faRotate} className='text-yellow-500' />
    )}
    {status === PromotionDirectiveStepStatus.SUCCESS && (
      <FontAwesomeIcon icon={faCheck} className='text-green-500' />
    )}
    {status === PromotionDirectiveStepStatus.FAILED && (
      <FontAwesomeIcon icon={faTimes} className='text-red-500' />
    )}
    {status === PromotionDirectiveStepStatus.SKIPPED && <FontAwesomeIcon icon={faBan} />}
    {status === PromotionDirectiveStepStatus.WONT_RUN && (
      <span className='size-3.5 rounded-full border border-dashed' />
    )}
  </Flex>
);
