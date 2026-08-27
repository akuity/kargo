import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Space, Tooltip } from 'antd';
import { ColumnType } from 'antd/es/table';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import {
  isPromotionWindowClosed,
  promotionWindowClosedMessage
} from '@ui/features/project/pipelines/promotion/promotion-window';
import { Stage } from '@ui/gen/api/v2/models';

import { ResumeAutoPromotionAction } from './resume-auto-promotion-action';

type Props = {
  onPromote: (stage: Stage) => void;
};

export const actionColumn = (props: Props): ColumnType<Stage> => ({
  render: (_, stage) => {
    if (isStageControlFlow(stage)) {
      return null;
    }

    // a closed promotion window forbids promotion of this Stage
    const windowClosed = isPromotionWindowClosed(stage);

    return (
      <Space size={6}>
        <ResumeAutoPromotionAction stage={stage} />
        <Tooltip title={windowClosed ? promotionWindowClosedMessage(stage) : undefined}>
          <Button
            onClick={() => props.onPromote(stage)}
            size='small'
            disabled={windowClosed}
            icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
          >
            Promote
          </Button>
        </Tooltip>
      </Space>
    );
  }
});
