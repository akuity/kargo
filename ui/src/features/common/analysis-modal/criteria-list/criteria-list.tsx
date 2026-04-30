import { faCheck, faRotateRight, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Space, Typography } from 'antd';
import classNames from 'classnames';

import { AnalysisStatus } from '../types';

import styles from './criteria-list.module.less';

const { Text } = Typography;

enum CriterionStatus {
  Fail = 'FAIL',
  Pass = 'PASS',
  InProgress = 'IN_PROGRESS',
  Pending = 'PENDING'
}

const defaultCriterionStatus = (analysisStatus: AnalysisStatus) =>
  analysisStatus === AnalysisStatus.Pending ? CriterionStatus.Pending : CriterionStatus.InProgress;

const criterionLabel = (measurementLabel: string, maxAllowed: number) =>
  maxAllowed === 0
    ? `No ${measurementLabel}.`
    : `Fewer than ${maxAllowed + 1} ${measurementLabel}.`;

interface CriteriaListItemProps {
  children: React.ReactNode;
  showIcon: boolean;
  status: CriterionStatus;
}

const CriteriaListItem = ({ children, showIcon, status }: CriteriaListItemProps) => {
  let StatusIcon: React.ReactNode | null = null;
  switch (status) {
    case CriterionStatus.Fail: {
      StatusIcon = <FontAwesomeIcon icon={faXmark} className='text-red-500' />;
      break;
    }
    case CriterionStatus.Pass: {
      StatusIcon = <FontAwesomeIcon icon={faCheck} className='text-green-500' />;
      break;
    }
    case CriterionStatus.InProgress: {
      StatusIcon = <FontAwesomeIcon icon={faRotateRight} className='text-blue-500' />;
      break;
    }
    case CriterionStatus.Pending:
    default: {
      break;
    }
  }

  return (
    <li>
      <Space size='small'>
        {showIcon && <>{StatusIcon}</>}
        {children}
      </Space>
    </li>
  );
};

interface CriteriaListProps {
  analysisStatus: AnalysisStatus;
  className?: string[] | string;
  consecutiveErrors: number;
  failures: number;
  inconclusives: number;
  maxConsecutiveErrors: number;
  maxFailures: number;
  maxInconclusives: number;
  showIcons: boolean;
}

export const CriteriaList = ({
  analysisStatus,
  className,
  consecutiveErrors,
  failures,
  inconclusives,
  maxConsecutiveErrors,
  maxFailures,
  maxInconclusives,
  showIcons
}: CriteriaListProps) => {
  let failureStatus = defaultCriterionStatus(analysisStatus);
  let errorStatus = defaultCriterionStatus(analysisStatus);
  let inconclusiveStatus = defaultCriterionStatus(analysisStatus);

  if (analysisStatus !== AnalysisStatus.Pending && analysisStatus !== AnalysisStatus.Running) {
    failureStatus = failures <= maxFailures ? CriterionStatus.Pass : CriterionStatus.Fail;
    errorStatus =
      consecutiveErrors <= maxConsecutiveErrors ? CriterionStatus.Pass : CriterionStatus.Fail;
    inconclusiveStatus =
      inconclusives <= maxInconclusives ? CriterionStatus.Pass : CriterionStatus.Fail;
  }

  return (
    <ul className={classNames(styles['criteria-list'], className)}>
      {maxFailures > -1 && (
        <CriteriaListItem status={failureStatus} showIcon={showIcons}>
          <Text>{criterionLabel('measurement failures', maxFailures)}</Text>
        </CriteriaListItem>
      )}
      {maxConsecutiveErrors > -1 && (
        <CriteriaListItem status={errorStatus} showIcon={showIcons}>
          <Text>{criterionLabel('consecutive measurement errors', maxConsecutiveErrors)}</Text>
        </CriteriaListItem>
      )}
      {maxInconclusives > -1 && (
        <CriteriaListItem status={inconclusiveStatus} showIcon={showIcons}>
          <Text>{criterionLabel('inconclusive measurements', maxInconclusives)}</Text>
        </CriteriaListItem>
      )}
    </ul>
  );
};
