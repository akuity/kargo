import {
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faCircleQuestion
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQuery } from '@tanstack/react-query';
import { Popover, Table, Tooltip, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import React from 'react';
import { useParams } from 'react-router-dom';

import { listPromotions } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Promotion } from '@ui/gen/v1alpha1/types_pb';

export const Promotions = () => {
  const { name: projectName, stageName } = useParams();
  const { data: promotionsResponse, isLoading } = useQuery({
    ...listPromotions.useQuery({ project: projectName, stage: stageName }),
    enabled: !!stageName
  });

  const promotions = React.useMemo(
    () =>
      // Immutable sorting
      [...(promotionsResponse?.promotions || [])].sort((a, b) => {
        if (!a.metadata?.creationTimestamp?.seconds || !b.metadata?.creationTimestamp?.seconds) {
          return 0;
        }

        if (a.metadata?.creationTimestamp?.seconds < b.metadata?.creationTimestamp?.seconds) {
          return 1;
        }

        if (a.metadata?.creationTimestamp?.seconds > b.metadata?.creationTimestamp?.seconds) {
          return -1;
        }

        return 0;
      }),
    [promotionsResponse]
  );

  const columns: ColumnsType<Promotion> = [
    {
      title: '',
      render: (_, promotion) => {
        switch (promotion.status?.phase) {
          case 'Succeeded':
            return (
              <Tooltip title='Succeeded' placement='right'>
                <FontAwesomeIcon
                  color={theme.defaultSeed.colorSuccess}
                  icon={faCircleCheck}
                  size='lg'
                />
              </Tooltip>
            );
          case 'Errored':
            return (
              <Popover content={promotion.status.error} title='Errored' placement='right'>
                <FontAwesomeIcon
                  color={theme.defaultSeed.colorError}
                  icon={faCircleExclamation}
                  size='lg'
                />
              </Popover>
            );
          case 'Pending':
          case 'Running':
            return (
              <Tooltip title={promotion.status?.phase} placement='right'>
                <FontAwesomeIcon icon={faCircleNotch} spin size='lg' />
              </Tooltip>
            );
          default:
            return (
              <Tooltip title={promotion.status?.phase || 'Unknown'} placement='right'>
                <FontAwesomeIcon color='#aaa' icon={faCircleQuestion} size='lg' />
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
      render: (_, promotion) => promotion.spec?.freight.substring(0, 7)
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
