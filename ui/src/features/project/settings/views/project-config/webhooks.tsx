import { faClipboard, faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, notification, Table, Typography } from 'antd';
import Card from 'antd/es/card/Card';
import classNames from 'classnames';
import { useState } from 'react';

import { WebhookReceiverDetails } from '@ui/gen/api/v1alpha1/generated_pb';

type WebhooksProps = {
  webhookReceivers: WebhookReceiverDetails[];
  className?: string;
};

export const Webhooks = (props: WebhooksProps) => {
  const webhookReceivers = props.webhookReceivers;

  return (
    <Card title='Webhooks' type='inner'>
      <Table
        pagination={{ defaultPageSize: 5, hideOnSinglePage: true }}
        className={classNames(props.className)}
        dataSource={webhookReceivers}
        columns={[
          {
            key: 'name',
            dataIndex: 'name',
            title: 'Webhook Name'
          },
          {
            key: 'url',
            title: 'Webhook URL',
            width: '60%',
            render: (_, record) => <WebhookURLColumn details={record} />
          }
        ]}
      />
    </Card>
  );
};

const WebhookURLColumn = (props: { details: WebhookReceiverDetails }) => {
  const [mask, setMask] = useState(true);

  return (
    <Flex gap={16} align='center'>
      <Typography.Text type='secondary' className={classNames({ 'text-xs': !mask })}>
        {mask ? '*********************' : props.details?.url}
      </Typography.Text>
      <Button
        size='small'
        icon={<FontAwesomeIcon icon={mask ? faEye : faEyeSlash} />}
        onClick={() => setMask(!mask)}
        type='text'
        className='ml-auto'
      />
      <Button
        size='small'
        icon={<FontAwesomeIcon icon={faClipboard} />}
        onClick={async () => {
          await navigator.clipboard.writeText(props.details?.url);
          notification.success({ message: 'URL copied.', placement: 'bottomRight' });
        }}
        type='text'
      />
    </Flex>
  );
};
