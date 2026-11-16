import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import classNames from 'classnames';
import { Link } from 'react-router-dom';

import { useToneColor } from '@ui/features/common/status-tone';

import styles from './stage-node.module.less';
import { StageWaiting } from './use-stage-waiting';

// StageNodeWaiting states what a Stage is waiting on in place of its phase.
// "Promoting" is true but useless once something has stopped: what the reader
// needs is which thing, and the way to go and unblock it.
//
// It takes the same tone as the phase it stands in for. A Stage waiting on a
// pull request is promoting -- the promotion is running, parked on a step --
// so a color of its own would imply a kind of state this is not. The wording
// carries the part the color would have. The mark is the one thing that does
// differ: the phase spins, and waiting is precisely what is not moving.
// An extension names its own waiting label and can make it any length, and
// the column it has to sit in is about 127px. Rather than cut the words off,
// the type steps down to fit: at the status size the column holds around 17
// characters, and each step buys roughly four more. The ellipsis stays as the
// last resort for a label too long for even the smallest step.
const labelSize = (label: string) => {
  if (label.length <= 17) {
    return undefined;
  }

  return label.length <= 20 ? 'sm' : 'xs';
};

const sizeClass = { sm: styles.statusTextSm, xs: styles.statusTextXs };

export const StageNodeWaiting = (props: { waiting: StageWaiting }) => {
  const { waiting } = props;

  const size = labelSize(waiting.label);
  const color = useToneColor('processing');

  const label = (
    <span
      className={classNames(styles.statusValue, 'min-w-0')}
      style={{ color }}
      title={waiting.title}
    >
      <FontAwesomeIcon icon={waiting.icon} className='text-[9px]' />
      <span className={classNames(styles.statusText, size && sizeClass[size])}>
        {waiting.label}
      </span>
    </span>
  );

  return (
    <>
      {waiting.to ? (
        <Link to={waiting.to} className={styles.statusValueLink}>
          {label}
        </Link>
      ) : (
        label
      )}

      {/* a second line only for somewhere outside Kargo to go */}
      {waiting.href && (
        <Typography.Link
          className={classNames(styles.statusLink, styles.statusLinkOwnLine)}
          href={waiting.href}
          target='_blank'
          rel='noreferrer'
          title={waiting.href}
        >
          {waiting.linkLabel}
          <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
        </Typography.Link>
      )}
    </>
  );
};
