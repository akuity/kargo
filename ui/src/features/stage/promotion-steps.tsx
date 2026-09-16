import { Alert, Collapse } from 'antd';
import { useMemo } from 'react';

import { Promotion } from '@ui/gen/api/v2/models';

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
    () => getPromotionOutputsByStepAlias(props.promotion) || {},
    [props.promotion]
  );

  const message = props.promotion?.status?.message;

  const phase = getPromotionStatusPhase(props.promotion);

  // A failed step's message becomes the Promotion's, so it is already on screen.
  const hasIndividualPromotionStepTerminalMessage = (
    props.promotion.status?.stepExecutionMetadata ?? []
  ).some((meta, i) => isFailedStep(i, props.promotion.status) && !!meta.message);

  let shouldShowMessage = false;

  if (isPromotionPhaseTerminal(phase)) {
    switch (phase) {
      case PromotionStatusPhase.FAILED:
      case PromotionStatusPhase.ERRORED:
        // The failing step already shows this message, so don't repeat it.
        shouldShowMessage = !hasIndividualPromotionStepTerminalMessage;
        break;
      case PromotionStatusPhase.ABORTED:
        // An abort is not attributable to any one step.
        shouldShowMessage = true;
        break;
    }
  }

  const steps = props.promotion?.spec?.steps ?? [];

  const items = steps.flatMap((step, i) => {
    const result = getPromotionDirectiveStepStatus(i, props.promotion.status);
    const key = step.as || `step-${i}`;
    const item = Step({ step, result, output: outputsByStepAlias[step.as || ''] });

    if (!isFailedStep(i, props.promotion.status)) {
      return [item];
    }

    const stepMessage = props.promotion.status?.stepExecutionMetadata?.[i]?.message;

    if (!stepMessage) {
      return [item];
    }

    return [
      { ...item, className: `${item.className || ''} !border-none` },
      {
        key: `${key}-error`,
        label: <Alert message={stepMessage} type='error' />,
        showArrow: false,
        collapsible: 'disabled' as const,
        styles: { header: { paddingTop: 0 } }
      }
    ];
  });

  return (
    <>
      <Collapse expandIconPosition='end' bordered={false} items={items} />
      {shouldShowMessage && !!message && (
        <Alert message={props.promotion.status?.message} type='error' className='mt-4' />
      )}
    </>
  );
};
