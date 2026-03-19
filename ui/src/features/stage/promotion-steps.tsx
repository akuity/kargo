import { Alert, Collapse } from 'antd';
import { useMemo } from 'react';

import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

import { getPromotionDirectiveStepStatus } from '../common/promotion-directive-step-status/utils';
import { PromotionStatusPhase, getPromotionStatusPhase } from '../common/promotion-status/utils';

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
      {!!props.promotion?.status?.message &&
        (phase === PromotionStatusPhase.FAILED || phase === PromotionStatusPhase.ERRORED ? (
          <Alert message={props.promotion?.status?.message} type='error' className='mt-4' />
        ) : (
          <div className='mt-4 rounded border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600'>
            {props.promotion.status.message}
          </div>
        ))}
    </>
  );
};
