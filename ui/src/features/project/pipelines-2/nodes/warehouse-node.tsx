import { faRefresh, faWarehouse } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Card, Flex } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import styles from './node-size-source-of-truth.module.less';

export const WarehouseNode = (props: { warehouse: Warehouse }) => {
  const warehouseState = useWarehouseState(props.warehouse);

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
      className={styles['warehouse-node-size']}
    >
      <center>
        <Button
          size='small'
          icon={<FontAwesomeIcon icon={faRefresh} />}
          loading={warehouseState.refreshing}
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
