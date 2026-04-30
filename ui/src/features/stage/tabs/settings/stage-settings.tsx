import { Card, Divider, Flex, Typography } from 'antd';

import { StageDeleteButton } from './stage-delete-button';
import { StageEditForm } from './stage-edit-form';

export const StageSettings = () => {
  return (
    <Flex gap={16} vertical>
      <Card title='General' type='inner'>
        <StageEditForm />
        <Divider />
        <Flex gap={16} align='center'>
          <Typography.Text strong>Delete Stage</Typography.Text>
          <StageDeleteButton />
        </Flex>
      </Card>
    </Flex>
  );
};
