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
import { Popover, Spin, Table, Tooltip, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React, { useEffect, useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';
import {
  getFreight,
  listPromotions
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/service/v1alpha1/service_connect';
import { ListPromotionsResponse } from '@ui/gen/service/v1alpha1/service_pb';
import { Freight, Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const Promotions = () => {
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
    return [...(promotionsResponse?.promotions || [])];
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
      title: 'Approved by',
      render: (_, promotion) => {
        const annotation = promotion.metadata?.annotations['kargo.akuity.io/create-actor'];
        const email = annotation ? annotation.split(':')[1] : 'N/A';
        return email;
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
