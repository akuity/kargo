import { createPromiseClient } from '@connectrpc/connect';
import { createConnectQueryKey, useQuery } from '@connectrpc/connect-query';
import {
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Popover, Table, Tooltip, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';

import { transportWithAuth } from '@ui/config/transport';
import { listPromotions } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { ListPromotionsResponse } from '@ui/gen/service/v1alpha1/service_pb';
import { Promotion } from '@ui/gen/v1alpha1/generated_pb';

import { sortPromotions } from './utils/sort';

export const Promotions = () => {
  const client = useQueryClient();
  const { name: projectName, stageName } = useParams();
  const { data: promotionsResponse, isLoading } = useQuery(
    listPromotions,
    { project: projectName, stage: stageName },
    { enabled: !!stageName }
  );

  useEffect(() => {
    if (isLoading || !promotionsResponse) {
      return;
    }
    const cancel = new AbortController();

    const watchPromotions = async () => {
      const promiseClient = createPromiseClient(KargoService, transportWithAuth);
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
        const listPromotionsQueryKey = createConnectQueryKey(listPromotions, {
          project: projectName,
          stage: stageName
        });
        client.setQueryData(listPromotionsQueryKey, { promotions });
      }
    };
    watchPromotions();

    return () => cancel.abort();
  }, [isLoading]);

  const promotions = React.useMemo(() => {
    // Immutable sorting
    return [...(promotionsResponse?.promotions || [])].sort(sortPromotions);
  }, [promotionsResponse]);

  const columns: ColumnsType<Promotion> = [
    {
      title: '',
      width: 24,
      render: (_, promotion) => {
        switch (promotion.status?.phase) {
          case 'Succeeded':
            return (
              <Popover
                content={promotion.status?.message}
                title={promotion.status?.phase}
                placement='right'
              >
                <FontAwesomeIcon
                  color={theme.defaultSeed.colorSuccess}
                  icon={faCircleCheck}
                  size='lg'
                />
              </Popover>
            );
          case 'Failed':
          case 'Errored':
            return (
              <Popover
                content={promotion.status?.message}
                title={promotion.status?.phase}
                placement='right'
              >
                <FontAwesomeIcon
                  color={theme.defaultSeed.colorError}
                  icon={faCircleExclamation}
                  size='lg'
                />
              </Popover>
            );
          case 'Running':
            return (
              <Tooltip title={promotion.status?.phase} placement='right'>
                <FontAwesomeIcon icon={faCircleNotch} spin size='lg' />
              </Tooltip>
            );
          case 'Pending':
          default:
            return (
              <Tooltip title='Pending' placement='right'>
                <FontAwesomeIcon color='#aaa' icon={faHourglassStart} size='lg' />
              </Tooltip>
            );
        }
      }
    },
    {
      title: 'Date',
      render: (_, promotion) => {
        const date = promotion.metadata?.creationTimestamp?.toDate();
        return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
      }
    },
    {
      title: 'Name',
      dataIndex: ['metadata', 'name']
    },
    {
      title: 'Freight',
      render: (_, promotion) => promotion.spec?.freight?.substring(0, 7)
    }
  ];

  return (
    <Table
      columns={columns}
      dataSource={promotions}
      size='small'
      pagination={{ hideOnSinglePage: true }}
      rowKey={(p) => p.metadata?.uid || ''}
      loading={isLoading}
    />
  );
};
