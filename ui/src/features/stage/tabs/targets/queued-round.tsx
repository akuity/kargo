import { faArrowRight, faHourglassStart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import { Link } from 'react-router-dom';

import { Round, roundPath } from '@ui/features/stage/utils/promotion-rollout';

import { queuedRoundWords } from './summary';

/**
 * QueuedRound says, in one short line above everything the tab says about the
 * Targets, that a round is waiting to start on the Stage.
 *
 * A queued round has attempted no Target, so the tiles and the squares go on
 * describing the last round that ran. This line stands where a Pending row
 * stands in a classic Stage's list of Promotions: a status, with the round one
 * click away, and no more said about it than that.
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
      {subject} is pending
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
