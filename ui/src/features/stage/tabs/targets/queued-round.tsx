import { faArrowRight, faHourglassStart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import { Link } from 'react-router-dom';

import { Round, roundPath } from '@ui/features/stage/utils/promotion-rollout';

import { queuedRoundWords } from './summary';

/**
 * QueuedRound names the round waiting to start on a Stage, on one line above
 * everything the tab says about the Targets.
 *
 * A queued round has attempted no Target, so the tiles and the squares go on
 * describing the last round that ran. This line is what tells a reader that
 * they do, and what is coming next.
 */
export const QueuedRound = ({
  project,
  stageName,
  queued,
  freightLabel
}: {
  project: string;
  stageName: string;
  queued: Round;
  freightLabel: (name: string) => string;
}) => {
  const link = roundPath(project, stageName, queued);
  const { subject, kind } = queuedRoundWords(queued, freightLabel);

  return (
    <Typography.Text type='secondary' className='text-xs'>
      <FontAwesomeIcon icon={faHourglassStart} className='mr-2' />
      {subject} is queued and has not started. The Targets below are shown as the last round to run
      left them.
      {link && (
        <>
          {' · '}
          <Link to={link}>
            view {kind}
            <FontAwesomeIcon icon={faArrowRight} className='ml-1' />
          </Link>
        </>
      )}
    </Typography.Text>
  );
};
