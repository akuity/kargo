import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Space } from 'antd';
import { ColumnType } from 'antd/es/table';

import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { ResumeAutoPromotionAction } from './resume-auto-promotion-action';

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
        <ResumeAutoPromotionAction stage={stage} />
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
