import { useMutation } from '@connectrpc/connect-query';
import { faRefresh, faWarehouse } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Card, Flex, message } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { refreshWarehouse } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import styles from './node-size-source-of-truth.module.less';

export const WarehouseNode = (props: { warehouse: Warehouse }) => {
  const navigate = useNavigate();

  const warehouseState = useWarehouseState(props.warehouse);

  const refreshWarehouseMutation = useMutation(refreshWarehouse, {
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
    }
  });

  return (
    <Card
      size='small'
      title={
        <Flex
          align='center'
          gap={16}
          className={classNames(warehouseState.hasError && 'text-red-500')}
        >
          <FontAwesomeIcon icon={faWarehouse} />
          <span className='text-xs'>{props.warehouse?.metadata?.name}</span>

          {warehouseState.hasError && <Badge status='error' />}
        </Flex>
      }
      className={(styles['warehouse-node-size'], 'cursor-pointer')}
      onClick={() =>
        navigate(
          generatePath(paths.warehouse, {
            name: props.warehouse?.metadata?.namespace,
            warehouseName: props.warehouse?.metadata?.name
          })
        )
      }
    >
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
