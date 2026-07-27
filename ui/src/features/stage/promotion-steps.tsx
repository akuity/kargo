import { Alert, Collapse } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Promotion } from '@ui/gen/api/v2/models';

import {
  getPromotionDirectiveStepStatus,
  isFailedStep,
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
  const { promotionStepExtensions } = useExtensionsContext();

  const [activeKeys, setActiveKeys] = useState<string[]>([]);

  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(props.promotion) || {},
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

  // Steps with a registered extension are interactive
  const hasExtension = (step: (typeof steps)[number]) =>
    promotionStepExtensions.some((ext) => ext.identifier === step.uses);

  // The first interactive step that's currently running, if any.
  let runningKey: string | undefined;

  const items = steps.flatMap((step, i) => {
    const result = getPromotionDirectiveStepStatus(i, props.promotion.status);
    const key = step.as || `step-${i}`;

    if (!runningKey && result === PromotionDirectiveStepStatus.RUNNING && hasExtension(step)) {
      runningKey = key;
    }

    const item = {
      ...Step({
        step,
        result,
        output: outputsByStepAlias[step.as || ''],
        promotion: props.promotion
      }),
      key
    };

    return isFailedStep(i, props.promotion.status)
      ? [{ ...item, className: `${item.className || ''} !border-none` }, errorItem]
      : [item];
  });

  useEffect(() => {
    const key = runningKey;
    if (key) {
      setActiveKeys((prev) => (prev.includes(key) ? prev : [...prev, key]));
    }
  }, [runningKey]);

  return (
    <>
      <Collapse
        expandIconPosition='end'
        bordered={false}
        items={items}
        activeKey={activeKeys}
        onChange={(keys) => setActiveKeys(typeof keys === 'string' ? [keys] : keys)}
      />
      {shouldShowMessage && (
        <Alert message={props.promotion.status?.message} type='error' className='mt-4' />
      )}
    </>
  );
};
