import { faArrowRotateLeft, faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Table, Tooltip } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useState } from 'react';
import { Link, generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  isPromotionRetryable
} from '@ui/features/common/promotion-status/utils';
import { getAlias, getShortFreightLabel } from '@ui/features/common/utils';
import { useWatchPromotions } from '@ui/features/project/pipelines/promotion/use-watch-promotions';
import { useListPromotions } from '@ui/gen/api/v2/core/core';
import { ArgoCDShard, Promotion } from '@ui/gen/api/v2/models';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { parseDate } from '@ui/utils/dates';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

const rollbackAnnotationKey = 'kargo.akuity.io/rollback';

export const Promotions = ({ argocdShard }: { argocdShard?: ArgoCDShard }) => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();

  const listPromotionsQuery = useListPromotions(
    projectName || '',
    { stage: stageName },
    { query: { enabled: !!stageName } }
  );

  const freightMap = useGetFreightMap(projectName || '');

  const onRetryPromotion = (promotion: Promotion) => {
    navigate(
      generatePath(paths.promote, {
        name: promotion?.metadata?.namespace || '',
        freight: promotion?.spec?.freight || '',
        stage: stageName || ''
      })
    );
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
      width: 56,
      render: (_, promotion) => {
        const promotionStatusPhase = getPromotionStatusPhase(promotion);
        const isAbortRequestPending =
          hasAbortRequest(promotion) && !isPromotionPhaseTerminal(promotionStatusPhase);
        const canRetry = isPromotionRetryable(promotionStatusPhase);
        const rollbackOrigin = promotion.metadata?.annotations?.[rollbackAnnotationKey];

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

            {rollbackOrigin && (
              <Tooltip title={`Rollback promotion: ${rollbackOrigin}`}>
                <FontAwesomeIcon icon={faArrowRotateLeft} className='text-xs text-gray-500' />
              </Tooltip>
            )}

            {canRetry && (
              <Tooltip title='Retry promotion'>
                <FontAwesomeIcon
                  className='text-xs cursor-pointer'
                  icon={faUndo}
                  onClick={() => onRetryPromotion(promotion)}
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
