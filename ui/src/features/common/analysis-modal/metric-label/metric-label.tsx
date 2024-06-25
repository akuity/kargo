import { Space } from 'antd';

import { StatusIndicator } from '../status-indicator/status-indicator';
import { AnalysisStatus, FunctionalStatus } from '../types';

import styles from './metric-label.module.less';

interface AnalysisModalProps {
  label: string;
  status: AnalysisStatus;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
}

export const MetricLabel = ({ label, status, substatus }: AnalysisModalProps) => (
  <Space size='small'>
    <StatusIndicator size='small' status={status} substatus={substatus} />
    <span className={styles['metric-label']} title={label}>
      {label}
    </span>
  </Space>
);
