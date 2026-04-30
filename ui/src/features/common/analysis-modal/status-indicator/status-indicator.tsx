import classNames from 'classnames';

import { AnalysisStatus, FunctionalStatus } from '../types';
import { statusIndicatorColors } from '../utils';

import styles from './status-indicator.module.less';

interface StatusIndicatorProps {
  children?: React.ReactNode;
  className?: string[] | string;
  size?: 'small' | 'large';
  status: AnalysisStatus;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
}

const substatusBackground = (status: FunctionalStatus) => {
  switch (status) {
    case FunctionalStatus.ERROR:
      return 'bg-red-400';
    case FunctionalStatus.WARNING:
      return 'bg-yellow-400';
    default:
      return '';
  }
};

export const StatusIndicator = ({
  children,
  className,
  size = 'large',
  status,
  substatus
}: StatusIndicatorProps) => {
  const mainPx = size === 'small' ? 14 : 28;
  const subPx = size === 'small' ? 8 : 12;
  const square = (px: number) => ({
    width: `${px}px`,
    height: `${px}px`,
    minHeight: `${px}px`
  });

  return (
    <div className={classNames('relative rounded-full', className)}>
      <div
        style={square(mainPx)}
        className={classNames(styles.indicator, statusIndicatorColors(status))}
      >
        {children}
      </div>
      {substatus !== undefined && (
        <div
          style={{ ...square(subPx) }}
          className={classNames(styles.substatus, substatusBackground(substatus))}
        />
      )}
    </div>
  );
};
