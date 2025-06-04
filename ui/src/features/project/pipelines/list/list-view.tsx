import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table, Typography } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { FreightArtifact } from '@ui/features/project/pipelines/freight/freight-artifact';
import {
  getLastPromotionDate,
  isStageControlFlow
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

type PipelineListViewProps = {
  stages: Stage[];
  warehouses: Warehouse[];
  className?: string;
};

export const PipelineListView = (props: PipelineListViewProps) => {
  const actionContext = useActionContext();

  return (
    <div className={classNames(props.className, 'px-2')}>
      <Table
        dataSource={props.stages}
        size='small'
        columns={[
          {
            title: 'Stage',
            width: '20%',
            render: (_, stage) => {
              return (
                <Link
                  to={generatePath(paths.stage, {
                    name: stage?.metadata?.namespace,
                    stageName: stage?.metadata?.name
                  })}
                >
                  {stage?.metadata?.name}
                  {isStageControlFlow(stage) ? (
                    <span className='text-xs ml-1'>(Control Flow)</span>
                  ) : (
                    ''
                  )}
                </Link>
              );
            }
          },
          {
            title: 'Phase',
            width: '10%',
            render: (_, stage) => {
              const stagePhase = getStagePhase(stage);

              if (getCurrentFreight(stage).length > 0) {
                const Comp = (
                  <Flex align='center' gap={4}>
                    {stagePhase}{' '}
                    <StageConditionIcon
                      conditions={stage?.status?.conditions || []}
                      noTooltip
                      className='text-[10px]'
                    />
                  </Flex>
                );

                if (stagePhase === 'Promoting') {
                  return (
                    <Link
                      to={generatePath(paths.promotion, {
                        name: stage?.metadata?.namespace,
                        promotionId: stage?.status?.currentPromotion?.name
                      })}
                    >
                      {Comp}
                    </Link>
                  );
                }

                return Comp;
              }

              return '-';
            }
          },
          {
            title: 'Health',
            width: '10%',
            render: (_, stage) => {
              const stageHealth = stage?.status?.health;

              if (stageHealth?.status) {
                return (
                  <Flex gap={4} align='center'>
                    {stageHealth?.status}
                    <HealthStatusIcon noTooltip health={stageHealth} />
                  </Flex>
                );
              }

              return '-';
            }
          },
          {
            title: 'Version',
            render: (_, stage) => {
              const currentFreight = getCurrentFreight(stage);

              // TODO: filter by sources
              const firstFreight = currentFreight[0];

              const totalArtifacts =
                (firstFreight?.commits?.length || 0) +
                (firstFreight?.charts?.length || 0) +
                (firstFreight?.images?.length || 0);

              return (
                <>
                  {firstFreight?.commits
                    ?.slice(0, 2)
                    .map((commit) => (
                      <FreightArtifact expand key={commit?.repoURL} artifact={commit} />
                    ))}

                  {firstFreight?.charts
                    ?.slice(0, 2)
                    .map((chart) => (
                      <FreightArtifact expand key={chart?.repoURL} artifact={chart} />
                    ))}

                  {firstFreight?.images
                    ?.slice(0, 2)
                    .map((image) => (
                      <FreightArtifact expand key={image?.repoURL} artifact={image} />
                    ))}

                  {totalArtifacts > 6 && (
                    <Typography.Text type='secondary' className='text-[10px]'>
                      +{' '}
                      {totalArtifacts -
                        (firstFreight?.charts?.slice(0, 2)?.length +
                          firstFreight?.commits?.slice(0, 2)?.length +
                          firstFreight?.images?.slice(0, 2)?.length)}{' '}
                      more
                    </Typography.Text>
                  )}
                </>
              );
            }
          },
          {
            title: 'Last Promotion',
            width: '15%',
            render: (_, stage) => {
              if (isStageControlFlow(stage)) {
                return '-';
              }

              const lastPromotion = getLastPromotionDate(stage);

              if (!lastPromotion) {
                return '-';
              }

              const date = timestampDate(lastPromotion) as Date;

              return (
                <Link
                  to={generatePath(paths.promotion, {
                    name: stage?.metadata?.namespace,
                    promotionId: stage?.status?.lastPromotion?.name
                  })}
                >
                  {formatDistance(date, new Date(), { addSuffix: true })}
                </Link>
              );
            }
          },
          {
            render: (_, stage) => {
              if (isStageControlFlow(stage)) {
                return null;
              }

              return (
                <Button
                  onClick={() => actionContext?.actPromote(IAction.PROMOTE, stage)}
                  size='small'
                  icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
                >
                  Promote
                </Button>
              );
            }
          }
        ]}
      />
    </div>
  );
};
