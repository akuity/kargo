import { useDndContext } from '@dnd-kit/core';
import { faTruckArrowRight, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import classNames from 'classnames';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

import styles from './drop-overlay.module.less';

type Props = {
  isOver: boolean;
  stage: Stage;
};

export const DropOverlay = ({ isOver, stage }: Props) => {
  const dndContext = useDndContext();

  const isDragging = Boolean(dndContext.active);

  // make sure that the freight can be promoted to this stage by checking the origin
  const eligible = Boolean(
    stage.spec?.requestedFreight?.find(
      (f) => f.origin?.name === dndContext.active?.data.current?.originName
    )
  );

  // Highlight every eligible stage while dragging so the user sees where the
  // freight can go. For ineligible stages we only surface the rejection when the
  // user actually hovers over one -- otherwise every unconnected stage would
  // flash a red "X" at once, which is noisier than it is helpful.
  const showPromote = isDragging && eligible;
  const showReject = isDragging && !eligible && isOver;

  const controlFlow = isStageControlFlow(stage);

  return (
    <div
      className={classNames(styles.dropOverlay, {
        [styles.hidden]: !showPromote && !showReject,
        [styles.reject]: showReject
      })}
      style={{ transform: isOver ? 'scale(0.96)' : undefined }}
    >
      {showReject ? (
        <>
          <FontAwesomeIcon icon={faXmark} />
          <Typography.Title level={5} className='!mb-0' style={{ color: 'inherit' }}>
            Can&apos;t promote here
          </Typography.Title>
        </>
      ) : (
        <>
          <FontAwesomeIcon icon={faTruckArrowRight} />
          <Typography.Title level={5} className='!mb-0'>
            Promote {controlFlow && 'to Downstream'}
          </Typography.Title>
        </>
      )}
    </div>
  );
};
