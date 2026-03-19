import { Alert, Collapse } from 'antd';
import { useMemo } from 'react';

import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

import { getPromotionDirectiveStepStatus } from '../common/promotion-directive-step-status/utils';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  PromotionStatusPhase
} from '../common/promotion-status/utils';

import { Step } from './promotion-step';
import { getPromotionOutputsByStepAlias } from './utils/promotion';

type PromotionStepsProps = {
  promotion: Promotion;
};

export const PromotionSteps = (props: PromotionStepsProps) => {
  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(props.promotion),
    [props.promotion]
  );

  const phase = getPromotionStatusPhase(props.promotion);

  const shouldShowMessage =
    isPromotionPhaseTerminal(phase) &&
    phase !== PromotionStatusPhase.SUCCEEDED &&
    !!props.promotion?.status?.message;

  return (
    <>
      <Collapse
        expandIconPosition='end'
        bordered={false}
        items={props.promotion?.spec?.steps.map((step, i) => {
          return Step({
            step,
            result: getPromotionDirectiveStepStatus(i, props.promotion.status),
            output: outputsByStepAlias?.[step?.as || '']
          });
        })}
      />
      {shouldShowMessage && (
        <Alert message={props.promotion?.status?.message} type='error' className='mt-4' />
      )}
    </>
  );
};
