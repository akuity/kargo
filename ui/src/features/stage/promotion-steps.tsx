import { Alert, Collapse, CollapseProps } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Promotion } from '@ui/gen/api/v2/models';

import {
  getPromotionDirectiveStepStatus,
  isFailedStep,
  isProgressingStep,
  isRetryingStep
} from '../common/promotion-directive-step-status/utils';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  PromotionStatusPhase
} from '../common/promotion-status/utils';

import { Step } from './promotion-step';
import { PromotionTaskGroupHeader } from './promotion-task-group-header';
import { getPromotionOutputsByStepAlias } from './utils/promotion';
import {
  groupPromotionSteps,
  IndexedPromotionStep,
  PromotionTaskGroup
} from './utils/promotion-step-groups';

type CollapseItem = NonNullable<CollapseProps['items']>[number];

type PromotionStepsProps = {
  promotion: Promotion;
};

export const PromotionSteps = (props: PromotionStepsProps) => {
  const { promotionStepExtensions } = useExtensionsContext();

  const [activeKeys, setActiveKeys] = useState<string[]>([]);

  const onActiveKeysChange = (keys: string | string[]) =>
    setActiveKeys(typeof keys === 'string' ? [keys] : keys);

  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(props.promotion) || {},
    [props.promotion]
  );

  const phase = getPromotionStatusPhase(props.promotion);

  const message = props.promotion?.status?.message;

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

  // Steps with a registered extension are interactive
  const hasExtension = (step: (typeof steps)[number]) =>
    promotionStepExtensions.some((ext) => ext.identifier === step.uses);

  const entries = groupPromotionSteps(steps, props.promotion?.spec?.inflatedTasks);

  // The first interactive step that's currently running, if any.
  let runningKey: string | undefined;
  let runningTaskKey: string | undefined;

  const toStepItems = ({ step, index: i }: IndexedPromotionStep): CollapseItem[] => {
    const result = getPromotionDirectiveStepStatus(i, props.promotion.status);
    const key = step.as || `step-${i}`;

    if (!runningKey && isProgressingStep(i, props.promotion.status) && hasExtension(step)) {
      runningKey = key;
    }

    const item = {
      ...Step({
        step,
        result,
        output: outputsByStepAlias[step.as || ''],
        promotion: props.promotion,
        stepIndex: i
      }),
      key
    };

    let alertType: 'error' | 'warning' | undefined;

    if (isFailedStep(i, props.promotion.status)) {
      alertType = 'error';
    } else if (isRetryingStep(i, props.promotion.status)) {
      alertType = 'warning';
    }

    const stepMessage = props.promotion.status?.stepExecutionMetadata?.[i]?.message;

    if (!alertType || !stepMessage) {
      return [item];
    }

    return [
      { ...item, className: `${item.className || ''} !border-none` },
      {
        key: `${key}-${alertType}`,
        label: <Alert title={stepMessage} type={alertType} />,
        showArrow: false,
        collapsible: 'disabled' as const,
        styles: { header: { paddingTop: 0 } }
      }
    ];
  };

  const toTaskItem = (group: PromotionTaskGroup): CollapseItem => {
    const stepItems = group.steps.flatMap(toStepItems);

    if (!runningTaskKey && stepItems.some((item) => item.key === runningKey)) {
      runningTaskKey = group.key;
    }

    return {
      key: group.key,
      className: 'border-gray-200 dark:border-neutral-700',
      label: <PromotionTaskGroupHeader group={group} promotion={props.promotion} />,
      children: (
        <Collapse
          expandIconPlacement='end'
          bordered={false}
          className='[&_.ant-collapse-title]:min-w-0'
          items={stepItems}
          activeKey={activeKeys}
          onChange={onActiveKeysChange}
        />
      )
    };
  };

  const items = entries.flatMap((entry) =>
    entry.type === 'task' ? [toTaskItem(entry)] : toStepItems(entry)
  );

  useEffect(() => {
    const keys = [runningTaskKey, runningKey].filter((key) => key !== undefined);
    setActiveKeys((prev) => [...prev, ...keys.filter((key) => !prev.includes(key))]);
  }, [runningKey, runningTaskKey]);

  return (
    <>
      <Collapse
        expandIconPlacement='end'
        bordered={false}
        // antd sizes the header text `flex: auto` with no min-width, so a long
        // step description can't truncate unless the wrapper may shrink.
        className='[&_.ant-collapse-title]:min-w-0'
        items={items}
        activeKey={activeKeys}
        onChange={onActiveKeysChange}
      />
      {shouldShowMessage && !!message && <Alert title={message} type='error' className='mt-4' />}
    </>
  );
};
