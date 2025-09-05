import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag } from 'antd';
import classNames from 'classnames';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionType } from '@ui/features/common/stage-status/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

type AnalysisRunLogsLinkProps = {
  stage: Stage;
  className?: string;
};

export const AnalysisRunLogsLink = (props: AnalysisRunLogsLinkProps) => {
  if (
    props.stage?.status?.conditions?.find(
      (condition) => condition?.type === StageConditionType.Promoting
    )
  ) {
    return null;
  }

  const recentVerification = props.stage?.status?.freightHistory?.[0]?.verificationHistory?.[0];

  const recentVerificationFailed = recentVerification?.phase === 'Failed';

  if (!recentVerificationFailed) {
    return null;
  }

  const logsLink = generatePath(paths.analysisRunLogs, {
    name: props.stage?.metadata?.namespace,
    stageName: props.stage?.metadata?.name,
    analysisRunId: recentVerification?.analysisRun?.name
  });

  return (
    <Link to={logsLink} target='_blank' className={classNames(props.className)}>
      <Tag color='orange' className='text-[10px]' bordered={false}>
        Analysis Run Logs <FontAwesomeIcon icon={faExternalLink} className='ml-1' />
      </Tag>
    </Link>
  );
};
