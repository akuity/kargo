import { createClient } from '@connectrpc/connect';
import { createConnectQueryKey, useQuery } from '@connectrpc/connect-query';
import { faArrowRotateLeft, faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Flex, Spin, Table, Tooltip } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useEffect, useState } from 'react';
import { Link, generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  isPromotionRetryable
} from '@ui/features/common/promotion-status/utils';
import {
  getFreight,
  listPromotions
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ListPromotionsResponse } from '@ui/gen/api/service/v1alpha1/service_pb';
import { KargoService } from '@ui/gen/api/service/v1alpha1/service_pb';
import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Promotion, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

const rollbackAnnotationKey = 'kargo.akuity.io/rollback';

type PromotionsProps = {
  argocdShard?: ArgoCDShard;
  stage?: Stage;
};

export const Promotions = ({ argocdShard, stage }: PromotionsProps) => {
  const client = useQueryClient();
  const navigate = useNavigate();

  const { name: projectName, stageName } = useParams();
  const { data: promotionsResponse, isLoading } = useQuery(
    listPromotions,
    { project: projectName, stage: stageName },
    { enabled: !!stageName }
  );

  const [curFreight, setCurFreight] = useState<string | undefined>();

  const { data: freightData, isLoading: isLoadingFreight } = useQuery(
    getFreight,
    { project: projectName, name: curFreight },
    {
      enabled: !!curFreight
    }
  );

  const onRetryPromotion = (promotion: Promotion) => {
    const stage = stageName;
    const project = promotion?.metadata?.namespace;
    const freight = promotion?.spec?.freight;

    navigate(
      generatePath(paths.promote, {
        name: project || '',
        freight: freight || '',
        stage: stage || ''
      })
    );
  };

  // modal kept in the same component for live view
  const [selectedPromotion, setSelectedPromotion] = useState<Promotion | undefined>();

  useEffect(() => {
    if (isLoading || !promotionsResponse) {
      return;
    }
    const cancel = new AbortController();

    const watchPromotions = async () => {
      const promiseClient = createClient(KargoService, transportWithAuth);
      const stream = promiseClient.watchPromotions(
        { project: projectName, stage: stageName },
        { signal: cancel.signal }
      );

      let promotions = (promotionsResponse as ListPromotionsResponse).promotions || [];

      for await (const e of stream) {
        const index = promotions?.findIndex(
          (item) => item.metadata?.name === e.promotion?.metadata?.name
        );
        if (e.type === 'DELETED') {
          if (index !== -1) {
            promotions = [...promotions.slice(0, index), ...promotions.slice(index + 1)];
          }
        } else {
          if (index === -1) {
            promotions = [...promotions, e.promotion as Promotion];
          } else {
            promotions = [
              ...promotions.slice(0, index),
              e.promotion as Promotion,
              ...promotions.slice(index + 1)
            ];
          }
        }

        // Update Promotions list
        const listPromotionsQueryKey = createConnectQueryKey({
          cardinality: 'finite',
          schema: listPromotions,
          input: {
            project: projectName,
            stage: stageName
          },
          transport: transportWithAuth
        });
        client.setQueryData(listPromotionsQueryKey, {
          promotions,
          $typeName: 'akuity.io.kargo.service.v1alpha1.ListPromotionsResponse'
        });
      }
    };
    watchPromotions();

    return () => cancel.abort();
  }, [isLoading]);

  const promotions = React.useMemo(() => {
    // Immutable sorting
    return [...(promotionsResponse?.promotions || [])].sort(promotionCompareFn);
  }, [promotionsResponse]);

  const rollbackPromotionNames = React.useMemo(
    () =>
      new Set(
        Object.values(stage?.status?.autoPromotionHolds || {})
          .map((hold) => hold?.promotionName)
          .filter(Boolean)
      ),
    [stage?.status?.autoPromotionHolds]
  );

  const columns: ColumnsType<Promotion> = [
    {
      title: '',
      width: 56,
      render: (_, promotion) => {
        const promotionStatusPhase = getPromotionStatusPhase(promotion);
        const isAbortRequestPending =
          hasAbortRequest(promotion) && !isPromotionPhaseTerminal(promotionStatusPhase);
        const canRetry = isPromotionRetryable(promotionStatusPhase);
        const isRollbackPromotion =
          promotion.metadata?.annotations?.[rollbackAnnotationKey] === 'true' ||
          rollbackPromotionNames.has(promotion.metadata?.name || '');

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
        const annotation = promotion.metadata?.annotations['kargo.akuity.io/create-actor'];
        const email = annotation ? annotation.split(':')[1] : 'N/A';

        return email || annotation;
      }
    },
    {
      title: 'Freight',
      render: (_, promotion) => (
        <Tooltip
          overlay={
            <Spin spinning={isLoadingFreight}>
              <div className='w-40 text-center truncate'>
                {(freightData?.result?.value as Freight)?.alias}
              </div>
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
        loading={isLoading}
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
