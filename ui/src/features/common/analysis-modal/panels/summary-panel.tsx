import { Typography } from 'antd';
import moment from 'moment';

import { Header } from '../header/header';
import { AnalysisStatus, FunctionalStatus } from '../types';

import styles from './styles.module.less';

const { Text } = Typography;

const timeRangeFormatter = (start: number, end?: number) => {
  const mStart = moment.unix(start);
  const startFormatted = mStart.format('LLL');
  if (!end) {
    return `${startFormatted} - present`;
  }
  const mEnd = moment.unix(end);
  const isSameDate = mStart.isSame(mEnd, 'day');
  const endFormatted = isSameDate ? mEnd.format('LT') : mEnd.format('LLL');
  return `${startFormatted} - ${endFormatted}`;
};

interface SummaryPanelProps {
  className?: string;
  endTime?: number;
  images: string[];
  message?: string;
  startTime?: number;
  status: AnalysisStatus;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
  title: string;
}

const SummarySection = ({ label, value }: { label: string; value: string }) => (
  <div className={styles['summary-section']}>
    <Text className='block' strong>
      {label}
    </Text>
    <Text>{value}</Text>
  </div>
);

export const SummaryPanel = ({
  className,
  endTime,
  images,
  message,
  startTime,
  status,
  substatus,
  title
}: SummaryPanelProps) => (
  <div className={className}>
    <Header className='mt-2 mb-6' title={title} status={status} substatus={substatus} />
    {images.length > 0 && (
      <SummarySection
        label={images.length > 1 ? `Versions` : `Version`}
        value={images.join(', ')}
      />
    )}
    {startTime !== null && (
      <SummarySection label='Runtime' value={timeRangeFormatter(startTime || 0, endTime || 0)} />
    )}
    {message && <SummarySection label='Summary' value={message} />}
  </div>
);
