import { faPause, faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Space } from 'antd';
import { ColumnType } from 'antd/es/table';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { AutoPromotionHoldsPopover } from '@ui/features/project/pipelines/promotion/auto-promotion-holds-popover';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

type Props = {
  onPromote: (stage: Stage) => void;
};

export const actionColumn = (props: Props): ColumnType<Stage> => ({
  render: (_, stage) => {
    if (isStageControlFlow(stage)) {
      return null;
    }

    return (
      <Space size={6}>
        <AutoPromotionHoldsPopover stage={stage} placement='bottomRight'>
          <Button size='small' icon={<FontAwesomeIcon icon={faPause} />}>
            Resume
          </Button>
        </AutoPromotionHoldsPopover>
        <Button
          onClick={() => props.onPromote(stage)}
          size='small'
          icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
        >
          Promote
        </Button>
      </Space>
    );
  }
});
