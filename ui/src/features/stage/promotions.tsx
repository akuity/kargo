import { createClient } from '@connectrpc/connect';
import { createConnectQueryKey, useMutation, useQuery } from '@connectrpc/connect-query';
import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Flex, Spin, Table, Tooltip } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useEffect, useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  isPromotionRetryable
} from '@ui/features/common/promotion-status/utils';
import {
  isExpiredResourceVersionError,
  isSameOrOlderResourceVersion
} from '@ui/features/utils/resource-version';
import {
  getFreight,
  listPromotions,
  promoteToStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ListPromotionsResponse } from '@ui/gen/api/service/v1alpha1/service_pb';
import { KargoService } from '@ui/gen/api/service/v1alpha1/service_pb';
import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Promotion } from '@ui/gen/api/v1alpha1/generated_pb';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { hasAbortRequest, promotionCompareFn } from './utils/promotion';

export const Promotions = ({ argocdShard }: { argocdShard?: ArgoCDShard }) => {
  const client = useQueryClient();

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

  const promotionMutation = useMutation(promoteToStage);

  const onRetryPromotion = (promotion: Promotion) => {
    const stage = stageName;
    const project = promotion?.metadata?.namespace;
    const freight = promotion?.spec?.freight;

    promotionMutation.mutate({
      stage,
      project,
      freight
    });
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
      const listPromotionsQueryKey = createConnectQueryKey({
        cardinality: 'finite',
        schema: listPromotions,
        input: {
          project: projectName,
          stage: stageName
        },
        transport: transportWithAuth
      });

      while (!cancel.signal.aborted) {
        const currentPromotionsResponse = client.getQueryData(listPromotionsQueryKey) as
          | ListPromotionsResponse
          | undefined;
        const stream = promiseClient.watchPromotions(
          {
            project: projectName,
            stage: stageName,
            resourceVersion: currentPromotionsResponse?.resourceVersion || ''
          },
          { signal: cancel.signal }
        );

        let promotions =
          currentPromotionsResponse?.promotions ||
          (promotionsResponse as ListPromotionsResponse).promotions ||
          [];

        try {
          for await (const e of stream) {
            const promotion = e.promotion;
            if (!promotion) {
              continue;
            }

            const index = promotions?.findIndex(
              (item) => item.metadata?.name === promotion.metadata?.name
            );
            if (e.type === 'DELETED') {
              if (index !== -1) {
                promotions = [...promotions.slice(0, index), ...promotions.slice(index + 1)];
              }
            } else if (
              e.type === 'ADDED' &&
              index !== -1 &&
              isSameOrOlderResourceVersion(promotions[index], promotion)
            ) {
              continue;
            } else {
              if (index === -1) {
                promotions = [...promotions, promotion];
              } else {
                promotions = [
                  ...promotions.slice(0, index),
                  promotion,
                  ...promotions.slice(index + 1)
                ];
              }
            }

            // Update Promotions list
            client.setQueryData(listPromotionsQueryKey, {
              promotions,
              resourceVersion:
                promotion.metadata?.resourceVersion ||
                (client.getQueryData(listPromotionsQueryKey) as ListPromotionsResponse)
                  ?.resourceVersion ||
                currentPromotionsResponse?.resourceVersion ||
                '',
              $typeName: 'akuity.io.kargo.service.v1alpha1.ListPromotionsResponse'
            });
          }
          return;
        } catch (err) {
          if (cancel.signal.aborted) {
            return;
          }
          if (isExpiredResourceVersionError(err)) {
            await client.refetchQueries({
              queryKey: listPromotionsQueryKey,
              exact: true
            });
            continue;
          }
          throw err;
        }
      }
    };
    watchPromotions().catch(() => undefined);

    return () => cancel.abort();
  }, [isLoading]);

  const promotions = React.useMemo(() => {
    // Immutable sorting
    return [...(promotionsResponse?.promotions || [])].sort(promotionCompareFn);
  }, [promotionsResponse]);

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
