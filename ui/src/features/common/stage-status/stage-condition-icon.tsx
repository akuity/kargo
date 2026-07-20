import {
  faCircle,
  faCircleMinus,
  faCircleNotch,
  faMagnifyingGlass,
  faTimesCircle,
  faSync,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo, useMemo } from 'react';

import {
  StageConditionType,
  StageConditionStatus,
  isTransientNotReadyReason
} from '@ui/features/common/stage-status/utils';
import { V1Condition } from '@ui/gen/api/v2/models';

import styles from './styles.module.less';
import { TruckIcon } from './truck-icon/truck-icon';

interface IconState {
  icon: IconDefinition;
  iconSpin?: boolean;
  tooltipTitle: string;
  tooltipMessage: string;
  iconClass: string;
}

export const StageConditionIcon = memo(
  ({
    conditions,
    className,
    noTooltip,
    isControllerDead,
    controllerName
  }: {
    conditions: V1Condition[];
    className?: string;
    noTooltip?: boolean;
    isControllerDead?: boolean;
    controllerName?: string;
  }) => {
    const { iconState, isPromoting } = useMemo(() => {
      // A dead (or absent) controller overrides everything: the
      // conditions on the Stage were written by the very controller that
      // has gone silent, so they cannot be trusted. Surface this as
      // Failed with a tooltip explaining why.
      if (isControllerDead) {
        return {
          iconState: {
            icon: faTimesCircle,
            tooltipTitle: 'Failed',
            tooltipMessage: controllerName
              ? `Controller '${controllerName}' is dead or nonexistent`
              : 'The default controller is dead or nonexistent',
            iconClass: 'text-red-400'
          },
          isPromoting: false
        };
      }

      const hasCondition = (
        type: StageConditionType,
        status: StageConditionStatus
      ): { condition: V1Condition | undefined; isActive: boolean } => {
        const condition = conditions.find((c) => c.type === type);
        return {
          condition,
          isActive: condition?.status === status
        };
      };

      const { condition: promotingCondition, isActive: isPromoting } = hasCondition(
        StageConditionType.Promoting,
        StageConditionStatus.True
      );

      const { condition: reconcilingCondition, isActive: isReconciling } = hasCondition(
        StageConditionType.Reconciling,
        StageConditionStatus.True
      );

      const { condition: readyCondition, isActive: isReady } = hasCondition(
        StageConditionType.Ready,
        StageConditionStatus.True
      );

      const isNotReady = readyCondition?.status === StageConditionStatus.False;

      // Not Ready for a transient reason (e.g. a rollout still in progress)
      // is not a failure.
      const isProgressing = isNotReady && isTransientNotReadyReason(readyCondition?.reason);
      const isFailed = isNotReady && !isProgressing;

      const { condition: verifiedCondition, isActive: isVerifying } = hasCondition(
        StageConditionType.Verified,
        StageConditionStatus.Unknown
      );

      // Default state
      let iconState: IconState = {
        icon: faCircleMinus,
        tooltipTitle: 'Unknown',
        tooltipMessage: '',
        iconClass: 'text-gray-400'
      };

      // Priority: Promoting > Verifying > Reconciling > Progressing > Failed > Ready
      if (isPromoting && promotingCondition?.reason !== 'NoFreight') {
        iconState = {
          icon: faCircleMinus,
          tooltipTitle: 'Promoting',
          tooltipMessage: promotingCondition?.message ?? '',
          iconClass: 'text-gray-400'
        };
      } else if (isVerifying && verifiedCondition?.reason !== 'NoFreight') {
        iconState = {
          icon: faMagnifyingGlass,
          tooltipTitle: 'Verifying',
          tooltipMessage: verifiedCondition?.message ?? '',
          iconClass: `text-blue-500 ${styles.magnifyingGlass}`
        };
      } else if (isReconciling) {
        iconState = {
          icon: faSync,
          tooltipTitle: 'Reconciling',
          tooltipMessage: reconcilingCondition?.message ?? '',
          iconClass: `text-yellow-500 ${styles.rotate}`
        };
      } else if (isProgressing) {
        iconState = {
          icon: faCircleNotch,
          iconSpin: true,
          tooltipTitle: 'Progressing',
          tooltipMessage:
            readyCondition?.message || 'Waiting for the Stage to reach a healthy state',
          iconClass: 'text-blue-500'
        };
      } else if (isFailed && readyCondition.reason !== 'NoFreight') {
        iconState = {
          icon: faTimesCircle,
          tooltipTitle: 'Failed',
          tooltipMessage: readyCondition?.message ?? '',
          iconClass: 'text-red-400'
        };
      } else if (isReady) {
        iconState = {
          icon: faCircle,
          tooltipTitle: 'Ready',
          tooltipMessage: readyCondition?.message ?? '',
          iconClass: 'text-green-400'
        };
      }

      return { iconState, isPromoting };
    }, [conditions, isControllerDead, controllerName]);

    const tooltipContent = useMemo(
      () => (
        <>
          {iconState.tooltipTitle && (
            <>
              <b>Stage Status:</b> {iconState.tooltipTitle}
            </>
          )}
          {iconState.tooltipMessage && <div>{iconState.tooltipMessage}</div>}
        </>
      ),
      [iconState.tooltipTitle, iconState.tooltipMessage] // Only recalculate when tooltip content changes
    );

    const Icon = isPromoting ? (
      <TruckIcon className={className} />
    ) : (
      <FontAwesomeIcon
        icon={iconState.icon}
        spin={iconState.iconSpin}
        className={classNames(className, iconState.iconClass)}
      />
    );

    if (noTooltip) {
      return Icon;
    }

    return <Tooltip title={tooltipContent}>{Icon}</Tooltip>;
  }
);
