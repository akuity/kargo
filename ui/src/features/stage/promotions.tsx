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
import { useWatchPromotions } from '@ui/features/project/pipelines/promotion/use-watch-promotions';
import { useListPromotions, usePromoteToStage } from '@ui/gen/api/v2/core/core';
import { ArgoCDShard, Promotion } from '@ui/gen/api/v2/models';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { parseDate } from '@ui/utils/dates';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

const rollbackAnnotationKey = 'kargo.akuity.io/rollback';

export const Promotions = ({ argocdShard }: { argocdShard?: ArgoCDShard }) => {
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
      render: (_, promotion) => {
        const annotation = promotion.metadata?.annotations?.['kargo.akuity.io/create-actor'];
        const email = annotation ? annotation.split(':')[1] : 'N/A';

        return email || annotation;
      }
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
