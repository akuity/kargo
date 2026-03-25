import { Alert, Collapse, ConfigProvider } from 'antd';
import { useMemo } from 'react';

import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
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
    !!props.promotion?.status?.message;

  const steps = props.promotion?.spec?.steps ?? [];

  const failedStepIndex = shouldShowMessage
    ? steps.findIndex(
        (_, i) =>
          getPromotionDirectiveStepStatus(i, props.promotion.status) ===
          PromotionDirectiveStepStatus.FAILED
      )
    : -1;

  const items = steps.flatMap((step, i) => {
    const result = getPromotionDirectiveStepStatus(i, props.promotion.status);
    const item = Step({ step, result, output: outputsByStepAlias?.[step?.as || ''] });

    if (shouldShowMessage && i === failedStepIndex) {
      return [
        item,
        {
          key: `error-${i}`,
          label: (
            <ConfigProvider
              theme={{ token: { colorErrorBorder: 'transparent', borderRadiusLG: 0 } }}
            >
              <Alert message={props.promotion?.status?.message} type='error' showIcon />
            </ConfigProvider>
          ),
          showArrow: false,
          collapsible: 'disabled' as const,
          style: { border: 'none' },
          styles: { header: { padding: 0, background: 'transparent' } }
        }
      ];
    }

    return [item];
  });

  return <Collapse expandIconPosition='end' items={items} />;
};
