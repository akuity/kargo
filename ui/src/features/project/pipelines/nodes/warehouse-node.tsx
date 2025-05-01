import { useMutation } from '@connectrpc/connect-query';
import {
  faArrowUpRightFromSquare,
  faMinus,
  faPlus,
  faRefresh,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Card, Flex, message } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { refreshWarehouse } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import styles from './node-size-source-of-truth.module.less';

export const WarehouseNode = (props: { warehouse: Warehouse }) => {
  const navigate = useNavigate();

  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const warehouseState = useWarehouseState(props.warehouse);

  const refreshWarehouseMutation = useMutation(refreshWarehouse, {
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
    }
  });

  const isSubscriptionHidden =
    freightTimelineControllerContext?.preferredFilter?.hideSubscriptions?.[
      props.warehouse?.metadata?.name || ''
    ];

  return (
    <Card
      size='small'
      title={
        <Flex justify='space-between'>
          <Flex
            align='center'
            gap={16}
            className={classNames(warehouseState.hasError && 'text-red-500')}
          >
            <FontAwesomeIcon icon={faWarehouse} />
            <span className='text-xs'>{props.warehouse?.metadata?.name}</span>

            {warehouseState.hasError && <Badge status='error' />}
          </Flex>
          <Button
            icon={<FontAwesomeIcon icon={faArrowUpRightFromSquare} />}
            size='small'
            onClick={() =>
              navigate(
                generatePath(paths.warehouse, {
                  name: props.warehouse?.metadata?.namespace,
                  warehouseName: props.warehouse?.metadata?.name
                })
              )
            }
          />
        </Flex>
      }
      className={(styles['warehouse-node-size'], 'relative')}
    >
      <Button
        size='small'
        icon={<FontAwesomeIcon icon={isSubscriptionHidden ? faPlus : faMinus} />}
        className='absolute -left-2 top-[50%] translate-y-[-50%] text-[10px]'
        onClick={(e) => {
          e.stopPropagation();

          const warehouseName = props.warehouse?.metadata?.name || '';
          const hiddenSubscriptions = {
            ...freightTimelineControllerContext?.preferredFilter.hideSubscriptions
          };

          if (isSubscriptionHidden) {
            delete hiddenSubscriptions[warehouseName];
          } else {
            hiddenSubscriptions[warehouseName] = true;
          }

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            hideSubscriptions: hiddenSubscriptions
          });
        }}
      />
      <center>
        <Button
          size='small'
          icon={<FontAwesomeIcon icon={faRefresh} />}
          loading={warehouseState.refreshing}
          onClick={(e) => {
            e.stopPropagation();
            refreshWarehouseMutation.mutate({
              project: props.warehouse?.metadata?.namespace,
              name: props.warehouse?.metadata?.name
            });
          }}
        >
          Refresh{warehouseState.refreshing && 'ing'}
        </Button>
      </center>
    </Card>
  );
};

const useWarehouseState = (warehouse: Warehouse) =>
  useMemo(() => {
    let refreshing = false;

    for (const condition of warehouse?.status?.conditions || []) {
      if (condition.type === 'Reconciling' && condition.status === 'True') {
        refreshing = true;
      }
    }

    let hasError = false;
    let notReady = false;
    let hasReconcilingCondition = false;

    for (const condition of warehouse?.status?.conditions || []) {
      if (condition.type === 'Healthy' && condition.status === 'False') {
        hasError = true;
      }

      if (condition.type === 'Reconciling') {
        hasReconcilingCondition = true;
      }

      if (condition.type === 'Ready' && condition.status === 'False') {
        notReady = true;
      }
    }

    if (notReady && !hasReconcilingCondition) {
      hasError = true;
    }

    return {
      refreshing,
      hasError
    };
  }, [warehouse]);
