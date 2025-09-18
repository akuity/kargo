import { useDndContext } from '@dnd-kit/core';
import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import classNames from 'classnames';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import styles from './drop-overlay.module.less';

type Props = {
  isOver: boolean;
  stage: Stage;
};

export const DropOverlay = ({ isOver, stage }: Props) => {
  const dndContext = useDndContext();
  const isDragging =
    Boolean(dndContext.active) &&
    // make sure that the freight can be promoted to this stage by checking the origin
    Boolean(
      stage.spec?.requestedFreight.find(
        (f) => f.origin?.name === dndContext.active?.data.current?.originName
      )
    );

  return (
    <div
      className={classNames(styles.dropOverlay, {
        [styles.hidden]: !isDragging
      })}
      style={{ transform: isOver ? 'scale(0.96)' : undefined }}
    >
      <>
        <FontAwesomeIcon icon={faTruckArrowRight} />
        <Typography.Title level={5} className='!mb-0'>
          Promote
        </Typography.Title>
      </>
    </div>
  );
};
