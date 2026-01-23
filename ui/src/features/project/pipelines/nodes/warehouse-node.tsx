import { useMutation } from '@connectrpc/connect-query';
import {
  faBarsStaggered,
  faCircleNotch,
  faMinus,
  faPlus,
  faRefresh,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Card, Flex, message, Tooltip } from 'antd';
import classNames from 'classnames';
import { useContext, useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { refreshResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import styles from './node-size-source-of-truth.module.less';

export const WarehouseNode = (props: { warehouse: WarehouseExpanded }) => {
  const colorContext = useContext(ColorContext);
  const navigate = useNavigate();

  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const warehouseState = useWarehouseState(props.warehouse);

  const refreshResourceTypeWarehouse = 'Warehouse';
  const refreshResourceMutation = useMutation(refreshResource, {
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
    }
  });

  const color = colorContext?.warehouseColorMap?.[props.warehouse?.metadata?.name || ''];

  const isSubscriptionHidden =
    freightTimelineControllerContext?.preferredFilter?.hideSubscriptions?.[
      props.warehouse?.metadata?.name || ''
    ];

  const isWarehouseNameLong = (props.warehouse?.metadata?.name?.length || 0) > 17;
  const warehouseNameHeader = `${props.warehouse?.metadata?.name?.slice(0, 17)}${isWarehouseNameLong ? '...' : ''}`;
  const WarehouseNameHeader = <span className='text-xs'>{warehouseNameHeader}</span>;

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
            {!warehouseState.reconciling && <FontAwesomeIcon icon={faWarehouse} />}

            {warehouseState.reconciling && (
              <FontAwesomeIcon icon={faCircleNotch} title='Reconciling' spin />
            )}

            {isWarehouseNameLong ? (
              <Tooltip title={props.warehouse?.metadata?.name}>{WarehouseNameHeader}</Tooltip>
            ) : (
              WarehouseNameHeader
            )}

            {warehouseState.hasError && <Badge status='error' />}
          </Flex>
          <Button
            icon={<FontAwesomeIcon icon={faBarsStaggered} />}
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
      className={styles['warehouse-node-size']}
      style={{
        border: color && `1px solid ${color}`
      }}
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
          onClick={(e) => {
            e.stopPropagation();
            refreshResourceMutation.mutate({
              project: props.warehouse?.metadata?.namespace,
              name: props.warehouse?.metadata?.name,
              resourceType: refreshResourceTypeWarehouse
            });
          }}
          loading={warehouseState.refreshing}
        >
          Refresh{warehouseState.refreshing && 'ing'}
        </Button>
      </center>
    </Card>
  );
};

const useWarehouseState = (warehouse: WarehouseExpanded) =>
  useMemo(() => {
    let reconciling = false;

    for (const condition of warehouse?.status?.conditions || []) {
      if (condition.type === 'Reconciling' && condition.status === 'True') {
        reconciling = true;
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

    const refreshAnnotation = warehouse?.metadata?.annotations?.['kargo.akuity.io/refresh'];
    const refreshing =
      typeof refreshAnnotation === 'string' &&
      warehouse?.metadata?.annotations?.['kargo.akuity.io/refresh'] !==
        warehouse?.status?.lastHandledRefresh;

    return {
      refreshing,
      reconciling,
      hasError
    };
  }, [warehouse]);
