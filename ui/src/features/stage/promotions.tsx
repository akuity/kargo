import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Spin, Table, Tooltip } from 'antd';
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
import { useWatchPromotions } from '@ui/features/project/pipelines/promotion/use-watch-promotions';
import { useGetFreight, useListPromotions, usePromoteToStage } from '@ui/gen/api/v2/core/core';
import { ArgoCDShard, Promotion } from '@ui/gen/api/v2/models';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

export const Promotions = ({ argocdShard }: { argocdShard?: ArgoCDShard }) => {
  const { name: projectName, stageName } = useParams();

  const listPromotionsQuery = useListPromotions(
    projectName || '',
    { stage: stageName },
    { query: { enabled: !!stageName } }
  );

  const [curFreight, setCurFreight] = useState<string | undefined>();

  const getFreightQuery = useGetFreight(projectName || '', curFreight || '', {
    query: { enabled: !!curFreight }
  });

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

  useWatchPromotions(projectName || '', stageName || '');

  const promotions = React.useMemo(() => {
    // Immutable sorting
    return [...(listPromotionsQuery.data?.data?.items || [])].sort(promotionCompareFn);
  }, [listPromotionsQuery?.data]);

  const columns: ColumnsType<Promotion> = [
    {
      title: '',
      width: 24,
      render: (_, promotion) => {
        const promotionStatusPhase = getPromotionStatusPhase(promotion);
        const isAbortRequestPending =
          hasAbortRequest(promotion) && !isPromotionPhaseTerminal(promotionStatusPhase);
        const canRetry = isPromotionRetryable(promotionStatusPhase);

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

            {canRetry && (
              <Tooltip title='Retry promotion'>
                <FontAwesomeIcon
                  className='text-xs cursor-pointer'
                  icon={faUndo}
                  onClick={() => !promotionMutation.isPending && onRetryPromotion(promotion)}
                />
              </Tooltip>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Date',
      render: (_, promotion) => {
        const date = timestampDate(promotion.metadata?.creationTimestamp);
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
      render: (_, promotion) => (
        <Tooltip
          overlay={
            <Spin spinning={getFreightQuery.isLoading}>
              <div className='w-40 text-center truncate'>{getFreightQuery.data?.data?.alias}</div>
            </Spin>
          }
          onOpenChange={() => {
            setCurFreight(promotion.spec?.freight);
          }}
        >
          <Link
            to={generatePath(paths.freight, {
              name: projectName,
              freightName: promotion.spec?.freight
            })}
          >
            {promotion.spec?.freight?.substring(0, 7)}
          </Link>
        </Tooltip>
      )
    },
    {
      title: '',
      render: (_, promotion, promotionIndex) => {
        const filteredUiPlugins = uiPlugins
          .filter((plugin) =>
            plugin.DeepLinkPlugin?.Promotion?.shouldRender({
              promotion,
              isLatestPromotion: promotionIndex === 0
            })
          )
          .map((plugin) => plugin.DeepLinkPlugin?.Promotion?.render);

        if (filteredUiPlugins?.length > 0) {
          return (
            <UiPluginHoles.DeepLinks.Promotion className='w-fit'>
              {filteredUiPlugins.map(
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
          );
        }

        return '-';
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
