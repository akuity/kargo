import { faMagnifyingGlassChart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Space, Typography } from 'antd';

import { StatusIndicator } from '../status-indicator/status-indicator';
import { AnalysisStatus, FunctionalStatus } from '../types';

const { Text, Title } = Typography;

interface HeaderProps {
  className?: string;
  status: AnalysisStatus;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
  subtitle?: string;
  title: string;
}

export const Header = ({ className, status, substatus, subtitle, title }: HeaderProps) => (
  <Space className={className} size='small' align='start'>
    <StatusIndicator size='large' status={status} substatus={substatus}>
      <FontAwesomeIcon icon={faMagnifyingGlassChart} />
    </StatusIndicator>
    <div>
      <Title level={4} className='m-0'>
        {title}
      </Title>
      {subtitle && <Text type='secondary'>{subtitle}</Text>}
    </div>
  </Space>
);
