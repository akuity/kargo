import { IconDefinition } from '@fortawesome/fontawesome-svg-core';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Spin, Tag } from 'antd';
import classNames from 'classnames';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PromotionStepStatusPhase } from '@ui/features/common/promotion-status/utils';
import { Stage } from '@ui/gen/api/v2/models';

import { useCurrentPromotion } from './stage-meta-utils';

type StepWaitingLabelProps = {
  stage: Stage;
  stepUses: string;
  label: string;
  icon: IconDefinition;
  className?: string;
};

// StepWaitingLabel renders a tag linking to the current promotion whenever a
// step of the given kind is pausing it awaiting user action.
export const StepWaitingLabel = (props: StepWaitingLabelProps) => {
  const { promotion, isFetching } = useCurrentPromotion(props.stage);

  if (isFetching) {
    return <Spin size='small' />;
  }

  // type safe
  if (!promotion || !promotion.spec || !promotion.spec.steps) {
    return null;
  }

  const isWaiting = promotion.spec.steps.some(
    (step: { uses?: string }, index: number) =>
      step?.uses === props.stepUses &&
      promotion?.status?.stepExecutionMetadata?.[index]?.status === PromotionStepStatusPhase.RUNNING
  );

  if (!isWaiting) {
    return null;
  }

  return (
    <Link
      to={generatePath(paths.promotion, {
        name: props.stage?.metadata?.namespace || '',
        promotionId: promotion?.metadata?.name || ''
      })}
    >
      <Tag color='blue' bordered={false} className={classNames(props.className)}>
        <span className='text-[8px]'>
          {props.label} <FontAwesomeIcon className='ml-1' icon={props.icon} />
        </span>
      </Tag>
    </Link>
  );
};
