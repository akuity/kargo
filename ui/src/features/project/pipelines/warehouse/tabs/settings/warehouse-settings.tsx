import { Card, Divider, Flex, Typography } from 'antd';

import { WarehouseDeleteButton } from './warehouse-delete-button';
import { WarehouseEditForm } from './warehouse-edit-form';

export const WarehouseSettings = () => {
  return (
    <Flex gap={16} vertical>
      <Card title='General' type='inner'>
        <WarehouseEditForm />
        <Divider />
        <Flex gap={16} align='center'>
          <Typography.Text strong>Delete Warehouse</Typography.Text>
          <WarehouseDeleteButton />
        </Flex>
      </Card>
    </Flex>
  );
};
