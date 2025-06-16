import { faClipboard, faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, notification, Table, Typography } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { parse } from 'yaml';

import { ProjectConfig, WebhookReceiverDetails } from '@ui/gen/api/v1alpha1/generated_pb';

type WebhooksProps = {
  projectConfigYAML: string;
  className?: string;
};

export const Webhooks = (props: WebhooksProps) => {
  const projectConfig = useMemo(
    () => parse(props.projectConfigYAML) as ProjectConfig,
    [props.projectConfigYAML]
  );

  // @ts-expect-error todo - update types in backend from 'receivers' to 'webhookReceivers'
  const webhookReceivers = (projectConfig?.status?.webhookReceivers ||
    []) as WebhookReceiverDetails[];

  return (
    <Table
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
