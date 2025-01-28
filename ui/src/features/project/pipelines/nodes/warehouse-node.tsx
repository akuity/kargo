import { useMutation } from '@connectrpc/connect-query';
import {
  faBuilding,
  faCircleNotch,
  faExclamationCircle,
  faFilter,
  faRefresh
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, message } from 'antd';
import classNames from 'classnames';
import { useContext } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { refreshWarehouse } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import { usePipelineContext } from '../context/use-pipeline-context';
import { MessageTooltip } from '../message-tooltip';
import { onError } from '../utils/util';

import styles from './custom-node.module.less';

type WarehouseNodeProps = {
  warehouse: Warehouse;
  warehouses?: number;
};

export const WarehouseNode = (props: WarehouseNodeProps) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const pipelineContext = usePipelineContext();

  const navigate = useNavigate();

  const refeshWarehouseMutation = useMutation(refreshWarehouse, {
    onError,
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
      pipelineContext?.state.clear();
      // TODO: refetchFreightData
    }
  });

  const warehouseName = props.warehouse?.metadata?.name || '';

  let refreshing = false;

  for (const condition of props.warehouse?.status?.conditions || []) {
    if (condition.type === 'Reconciling' && condition.status === 'True') {
      refreshing = true;
    }
  }

  let hasError = false;
  let notReady = false;
  let hasReconcilingCondition = false;
  let errMessage = '';

  for (const condition of props.warehouse?.status?.conditions || []) {
    if (condition.type === 'Healthy' && condition.status === 'False') {
      hasError = true;
      errMessage = condition.message || '';
    }

    if (condition.type === 'Reconciling') {
      hasReconcilingCondition = true;
    }

    if (condition.type === 'Ready' && condition.status === 'False') {
      notReady = true;
      errMessage = condition.message || '';
    }
  }

  if (notReady && !hasReconcilingCondition) {
    hasError = true;
  }

  return (
    <div
      className={classNames(styles.warehouseNode)}
      onClick={() =>
        navigate(
          generatePath(paths.warehouse, {
            name: pipelineContext?.project,
            warehouseName: props.warehouse?.metadata?.name
          })
        )
      }
    >
      <div className={classNames(styles.header)}>
        <h3>{warehouseName}</h3>

        <div className='ml-auto space-x-2'>
          {refreshing && <FontAwesomeIcon icon={faCircleNotch} spin />}
          {/* TODO: make this error tooltip beautiful */}
          {hasError && (
            <MessageTooltip
              message={
                <div className='flex flex-col gap-4 overflow-y-scroll text-wrap max-h-48'>
                  <div
                    className='cursor-pointer min-w-0'
                    onClick={() => {
                      if (message) {
                        navigator.clipboard.writeText(errMessage);
                      }
                    }}
                  >
                    <div className='flex text-wrap'>
                      <FontAwesomeIcon icon={faExclamationCircle} className='mr-2 mt-1 pl-1' />
                      {errMessage}
                    </div>
                  </div>
                </div>
              }
              icon={faExclamationCircle}
              iconClassName='text-red-500'
            />
          )}
          <FontAwesomeIcon
            icon={faBuilding}
            className='text-base'
            style={{
              color: warehouseColorMap[warehouseName]
            }}
          />
        </div>
      </div>

      <div className={classNames(styles.body, 'flex')}>
        {(props.warehouses || 0) > 1 && (
          <Button
            icon={<FontAwesomeIcon icon={faFilter} />}
            size='small'
            type={
              pipelineContext?.selectedWarehouse === props.warehouse?.metadata?.name
                ? 'primary'
                : 'default'
            }
            onClick={(e) => {
              e.stopPropagation();

              const newSelectedWarehouse =
                pipelineContext?.selectedWarehouse === props.warehouse?.metadata?.name
                  ? ''
                  : props.warehouse?.metadata?.name;

              pipelineContext?.setSelectedWarehouse(newSelectedWarehouse || '');
            }}
          />
        )}
        <Button
          icon={<FontAwesomeIcon icon={faRefresh} />}
          size='small'
          className='mx-auto'
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            refeshWarehouseMutation.mutate({
              name: props.warehouse?.metadata?.name,
              project: pipelineContext?.project
            });
          }}
        >
          Refresh
        </Button>
      </div>
    </div>
  );
};
