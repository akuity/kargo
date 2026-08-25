import {
  faQuestionCircle,
  faTrash,
  faWandMagicSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Typography } from 'antd';

import { CreateWarehouseWizard } from '@ui/features/stage/create-warehouse/create-warehouse-wizard';

import { WarehouseDraft, exampleWarehouse, initialWarehouse } from '../types';

type StepWarehousesProps = {
  value: WarehouseDraft[];
  onChange: (value: WarehouseDraft[]) => void;
};

export const StepWarehouses = ({ value, onChange }: StepWarehousesProps) => {
  const add = () => onChange([...value, initialWarehouse()]);
  const loadExample = () => onChange([...value, exampleWarehouse()]);

  return (
    <Card
      type='inner'
      title={
        <Space size={4}>
          Warehouses
          <Popover content='A Warehouse subscribes to container image, Git, and Helm chart repositories and assembles new artifacts into Freight.'>
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      extra={
        <Button icon={<FontAwesomeIcon icon={faWarehouse} />} onClick={add}>
          Add Warehouse
        </Button>
      }
    >
      {value.length === 0 ? (
        <div className='flex flex-col items-center py-10 text-center'>
          <FontAwesomeIcon
            icon={faWarehouse}
            className='text-2xl text-gray-300 dark:text-neutral-600 mb-3'
          />
          <div className='text-sm text-gray-500 dark:text-neutral-400 max-w-md mb-5'>
            Add a Warehouse to watch a repository for new artifacts. Not sure where to start? Load
            the example, which watches the public guestbook image, inspired by akuity&apos;s{' '}
            <code>kargo-simple</code> example. You can also continue and add Warehouses later.
          </div>
          <Space>
            <Button
              type='primary'
              icon={<FontAwesomeIcon icon={faWandMagicSparkles} />}
              onClick={loadExample}
            >
              Load example
            </Button>
            <Button icon={<FontAwesomeIcon icon={faWarehouse} />} onClick={add}>
              Add Warehouse
            </Button>
          </Space>
        </div>
      ) : (
        <Flex gap={16} vertical>
          {value.map((warehouse, i) => (
            <Card
              key={i}
              title={
                <span className='flex items-center gap-2'>
                  <FontAwesomeIcon
                    icon={faWarehouse}
                    className='text-gray-400 dark:text-neutral-500'
                  />
                  <span className='font-mono text-sm'>{warehouse.name || '(unnamed)'}</span>
                </span>
              }
              extra={
                <Button
                  type='text'
                  danger
                  size='small'
                  icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                  onClick={() => onChange(value.filter((_, x) => x !== i))}
                />
              }
            >
              <CreateWarehouseWizard
                alwaysShowDescription
                formState={{ name: warehouse.name, spec: warehouse.spec }}
                setFormState={(next) => {
                  const n = next as { name?: string; spec?: Record<string, unknown> };
                  onChange(
                    value.map((w, x) =>
                      x === i ? { name: n.name ?? '', spec: n.spec ?? { subscriptions: [] } } : w
                    )
                  );
                }}
              />
            </Card>
          ))}
        </Flex>
      )}
    </Card>
  );
};
