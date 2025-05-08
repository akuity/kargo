import {
  faCircle,
  faCircleMinus,
  faMagnifyingGlass,
  faTimesCircle,
  faSync,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo, useMemo } from 'react';

import { StageConditionType, StageConditionStatus } from '@ui/features/common/stage-status/utils';
import type { Condition } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

import styles from './styles.module.less';
import { TruckIcon } from './truck-icon/truck-icon';

interface IconState {
  icon: IconDefinition;
  tooltipTitle: string;
  tooltipMessage: string;
  iconClass: string;
}

export const StageConditionIcon = memo(
  ({
    conditions,
    className,
    noTooltip
  }: {
    conditions: Condition[];
    className?: string;
    noTooltip?: boolean;
  }) => {
    const { iconState, isPromoting } = useMemo(() => {
      const hasCondition = (
        type: StageConditionType,
        status: StageConditionStatus
      ): { condition: Condition | undefined; isActive: boolean } => {
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

      const isFailed = readyCondition?.status === StageConditionStatus.False;

      const { condition: verifiedCondition, isActive: isVerifying } = hasCondition(
        StageConditionType.Verified,
        StageConditionStatus.True
      );

      // Default state
      let iconState: IconState = {
        icon: faCircleMinus,
        tooltipTitle: '',
        tooltipMessage: '',
        iconClass: 'text-gray-400'
      };

      // Priority: Promoting > Verifying > Reconciling > Failed > Ready
      if (isPromoting) {
        iconState = {
          icon: faCircleMinus,
          tooltipTitle: 'Promoting',
          tooltipMessage: promotingCondition?.message ?? '',
          iconClass: 'text-gray-400'
        };
      } else if (isVerifying) {
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
      } else if (isFailed) {
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
    }, [conditions]); // Only recalculate when conditions changes

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
        className={classNames(className, iconState.iconClass)}
      />
    );

    if (noTooltip) {
      return Icon;
    }

    return <Tooltip title={tooltipContent}>{Icon}</Tooltip>;
  }
);
