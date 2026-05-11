import { Alert, Collapse } from 'antd';
import { useMemo } from 'react';

import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

import {
  getPromotionDirectiveStepStatus,
  isFailedStep
} from '../common/promotion-directive-step-status/utils';
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
    phase !== PromotionStatusPhase.ERRORED && // because its already handled at individual step level
    !!props.promotion?.status?.message;

  const steps = props.promotion?.spec?.steps ?? [];

  const errorItem = {
    key: 'error',
    label: <Alert message={props.promotion.status?.message} type='error' />,
    showArrow: false,
    collapsible: 'disabled' as const,
    styles: { header: { paddingTop: 0 } }
  };

  const items = steps.flatMap((step, i) => {
    const result = getPromotionDirectiveStepStatus(i, props.promotion.status);
    const item = Step({ step, result, output: outputsByStepAlias[step.as || ''] });

    return isFailedStep(i, props.promotion.status)
      ? [{ ...item, className: `${item.className || ''} !border-none` }, errorItem]
      : [item];
  });

  return (
    <>
      <Collapse expandIconPosition='end' bordered={false} items={items} />
      {shouldShowMessage && (
        <Alert message={props.promotion.status?.message} type='error' className='mt-4' />
      )}
    </>
  );
};
