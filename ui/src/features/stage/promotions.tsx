import { faBackwardStep, faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table, Tooltip, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  isPromotionRetryable
} from '@ui/features/common/promotion-status/utils';
import { getAlias, getShortFreightLabel } from '@ui/features/common/utils';
import { getPromotionActor } from '@ui/features/project/pipelines/promotion/get-promotion-actor';
import { PromotionWindowAlert } from '@ui/features/project/pipelines/promotion/promotion-window-alert';
import { useWatchPromotions } from '@ui/features/project/pipelines/promotion/use-watch-promotions';
import { useListPromotions, usePromoteToStage } from '@ui/gen/api/v2/core/core';
import { ArgoCDShard, Promotion, Stage } from '@ui/gen/api/v2/models';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { parseDate } from '@ui/utils/dates';

import { isStageTargetAware } from '../project/pipelines/nodes/stage-meta-utils';
import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { PromotionRolloutDetails } from './promotion-rollout/promotion-rollout-details';
import { PromotionRollouts } from './promotion-rollouts';
import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

const rollbackAnnotationKey = 'kargo.akuity.io/rollback';

/**
 * Promotions is the Stage drawer's Promotions tab, which shows a different
 * thing depending on how the Stage promotes.
 *
 * A classic Stage promotes to itself, with one Promotion per round, so its tab
 * lists Promotions. A target-aware Stage promotes to the Targets it governs by
 * way of a PromotionRollout that fans out into one Promotion per Target, so
 * its tab lists the rollouts and leaves the children to be read inside one.
 * Listing both at this level would bury a fleet's rounds under hundreds of
 * Promotions that say nothing on their own.
 */
export const Promotions = ({ stage, argocdShard }: { stage: Stage; argocdShard?: ArgoCDShard }) =>
  isStageTargetAware(stage) ? (
    <TargetAwarePromotions stage={stage} />
  ) : (
    <ClassicPromotions stage={stage} argocdShard={argocdShard} />
  );

const TargetAwarePromotions = ({ stage }: { stage: Stage }) => {
  const { name: projectName, stageName, promotionRolloutName } = useParams();

  // One round, opened from the list, replaces the list in place rather than
  // opening a second drawer over the Stage's: a round is a thing the Stage
  // did, and reading it should not mean leaving the Stage.
  if (promotionRolloutName) {
    return (
      <PromotionRolloutDetails
        // Retrying moves from one round straight to another, which would
        // otherwise keep the chips and search of the round left behind and
        // open the new one already filtered.
        key={promotionRolloutName}
        project={projectName || ''}
        stageName={stageName || ''}
        promotionRolloutName={promotionRolloutName}
      />
    );
  }

  return (
    <>
      <PromotionWindowAlert stage={stage} className='mb-4' />
      <PromotionRollouts projectName={projectName || ''} stageName={stageName || ''} />
    </>
  );
};

const ClassicPromotions = ({ stage, argocdShard }: { stage: Stage; argocdShard?: ArgoCDShard }) => {
  const { token } = theme.useToken();

  const { name: projectName, stageName } = useParams();

  const listPromotionsQuery = useListPromotions(
    projectName || '',
    { stage: stageName },
    { query: { enabled: !!stageName } }
  );

  const freightMap = useGetFreightMap(projectName || '');

  const promotionMutation = usePromoteToStage();

  const onRetryPromotion = (promotion: Promotion) => {
    promotionMutation.mutate({
      project: promotion?.metadata?.namespace || '',
      stage: stageName || '',
      data: { freight: promotion?.spec?.freight }
    });
  };

  // modal kept in the same component for live view
  const [selectedPromotion, setSelectedPromotion] = useState<Promotion | undefined>();

  useWatchPromotions(projectName || '', stageName || '', !listPromotionsQuery.isLoading);

  const promotions = React.useMemo(() => {
    // Immutable sorting
    return [...(listPromotionsQuery.data?.data?.items || [])].sort(promotionCompareFn);
  }, [listPromotionsQuery?.data]);

  const columns: ColumnsType<Promotion> = [
    {
      title: '',
      width: 40,
      render: (_, promotion) => {
        const promotionStatusPhase = getPromotionStatusPhase(promotion);
        const isAbortRequestPending =
          hasAbortRequest(promotion) && !isPromotionPhaseTerminal(promotionStatusPhase);
        const isRollbackPromotion =
          promotion.metadata?.annotations?.[rollbackAnnotationKey] === 'true';

        // generally controller quickly Abort promotion
        // but incase if controller is off for some reason, this messaging ensures accurate information
        if (isAbortRequestPending && promotion?.status) {
          promotion.status.message = 'Promotion Abort Request is in Queue';
        }

        return (
          <Flex gap={8} align='center'>
            <PromotionStatusIcon
              status={promotion.status}
              color={isAbortRequestPending ? 'red' : ''}
            />

            {isRollbackPromotion && (
              <Tooltip title='Rollback promotion'>
                <FontAwesomeIcon icon={faBackwardStep} style={{ color: token.colorTextTertiary }} />
              </Tooltip>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Date',
      render: (_, promotion) => {
        const date = parseDate(promotion.metadata?.creationTimestamp);
        return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
      }
    },
    {
      title: 'Name',
      render: (_, promotion) => (
        <a onClick={() => setSelectedPromotion(promotion)}>{promotion.metadata?.name}</a>
      )
    },
    {
      title: 'Created By',
      render: (_, promotion) => getPromotionActor(promotion)
    },
    {
      title: 'Freight',
      render: (_, promotion) => {
        const freightName = promotion.spec?.freight || '';

        return (
          <Link
            to={generatePath(paths.freight, {
              name: projectName,
              freightName
            })}
          >
            {getShortFreightLabel(freightName, getAlias(freightMap[freightName]))}
          </Link>
        );
      }
    },
    {
      title: '',
      align: 'right',
      render: (_, promotion, promotionIndex) => {
        const canRetry = isPromotionRetryable(getPromotionStatusPhase(promotion));

        const deepLinks = uiPlugins
          .filter((plugin) =>
            plugin.DeepLinkPlugin?.Promotion?.shouldRender({
              promotion,
              isLatestPromotion: promotionIndex === 0
            })
          )
          .map((plugin) => plugin.DeepLinkPlugin?.Promotion?.render);

        return (
          <Flex gap={8} align='center' justify='flex-end'>
            {deepLinks.length > 0 && (
              <UiPluginHoles.DeepLinks.Promotion className='w-fit'>
                {deepLinks.map(
                  (ApplyPlugin, idx) =>
                    ApplyPlugin && (
                      <ApplyPlugin
                        key={idx}
                        promotion={promotion}
                        isLatestPromotion={promotionIndex === 0}
                        unstable_argocdShardUrl={argocdShard?.url}
                      />
                    )
                )}
              </UiPluginHoles.DeepLinks.Promotion>
            )}

            {canRetry && (
              <Button
                size='small'
                icon={<FontAwesomeIcon icon={faUndo} />}
                loading={promotionMutation.isPending}
                onClick={() => onRetryPromotion(promotion)}
              >
                Retry
              </Button>
            )}
          </Flex>
        );
      }
    }
  ];

  return (
    <>
      <PromotionWindowAlert stage={stage} className='mb-4' />

      <Table
        columns={columns}
        dataSource={promotions}
        size='small'
        pagination={{ hideOnSinglePage: true }}
        rowKey={(p) => p.metadata?.uid || ''}
        loading={listPromotionsQuery.isLoading}
      />

      {selectedPromotion && (
        <PromotionComponent
          visible={!!selectedPromotion}
          hide={() => setSelectedPromotion(undefined)}
          promotionId={selectedPromotion?.metadata?.name || ''}
          project={projectName || ''}
        />
      )}
    </>
  );
};
