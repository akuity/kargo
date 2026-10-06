import { useDndContext } from '@dnd-kit/core';
import { faCalendarXmark, faTruckArrowRight, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Typography } from 'antd';
import classNames from 'classnames';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

import { isPromotionWindowClosed, promotionWindowReopensLabel } from '../promotion-window';

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

  const windowClosed = isPromotionWindowClosed(stage);

  // Highlight every eligible stage while dragging so the user sees where the
  // freight can go. For ineligible stages we only surface the rejection when the
  // user actually hovers over one -- otherwise every unconnected stage would
  // flash a red "X" at once, which is noisier than it is helpful.
  //
  // A closed promotion window is different: the stage is connected, so the user
  // would otherwise aim for it. Mark it as blocked for the whole drag.
  const showBlocked = isDragging && eligible && windowClosed;
  const showPromote = isDragging && eligible && !windowClosed;
  const showReject = isDragging && !eligible && isOver;

  const controlFlow = isStageControlFlow(stage);

  let content = (
    <>
      <FontAwesomeIcon icon={faTruckArrowRight} />
      <Typography.Title level={5} className='!mb-0'>
        Promote {controlFlow && 'to Downstream'}
      </Typography.Title>
    </>
  );

  if (showBlocked) {
    content = (
      // the icon tracks the title rather than the middle of both lines
      <Flex gap={8} align='baseline'>
        <FontAwesomeIcon icon={faCalendarXmark} className='relative top-[2px]' />
        <Flex vertical align='center'>
          <Typography.Title level={5} className='!mb-0'>
            Promotion window closed
          </Typography.Title>
          <Typography.Text type='secondary' className='text-[10px]'>
            {promotionWindowReopensLabel(stage)}
          </Typography.Text>
        </Flex>
      </Flex>
    );
  } else if (showReject) {
    content = (
      <>
        <FontAwesomeIcon icon={faXmark} />
        <Typography.Title level={5} className='!mb-0' style={{ color: 'inherit' }}>
          Can&apos;t promote here
        </Typography.Title>
      </>
    );
  }

  return (
    <div
      className={classNames(styles.dropOverlay, {
        [styles.hidden]: !showPromote && !showReject && !showBlocked,
        [styles.reject]: showReject
      })}
      style={{ transform: isOver ? 'scale(0.96)' : undefined }}
    >
      {content}
    </div>
  );
};
