import {
  faClipboard,
  faEye,
  faEyeSlash,
  faPencil,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, notification, Table, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { useState } from 'react';

import { WebhookReceiverConfig, WebhookReceiverDetails } from '@ui/gen/api/v2/models';

import {
  isTypelessReceiver,
  receiverSecretName,
  receiverType,
  receiverTypeKey,
  receiverTypeLabel
} from './webhook-receiver-form-utils';

const ReceiverCell = ({ config }: { config: WebhookReceiverConfig }) => {
  const key = receiverTypeKey(config);
  const type = receiverType(key);

  return (
    <Flex align='center' gap={8}>
      {type?.icon && <FontAwesomeIcon icon={type.icon} />}
      <Typography.Text>{receiverTypeLabel(key)}</Typography.Text>
    </Flex>
  );
};

const URLCell = ({ url }: { url?: string }) => {
  const [masked, setMasked] = useState(true);

  // the controller mints the URL from the Secret, so it appears only once the
  // config has been reconciled -- a receiver saved moments ago has none yet
  if (!url) {
    return (
      <Tooltip title='Kargo assigns the URL once it has reconciled the config. Refresh to check again.'>
        <Typography.Text type='secondary' italic>
          Pending
        </Typography.Text>
      </Tooltip>
    );
  }

  return (
    <Flex gap={8} align='center'>
      {/* a receiver URL is one long unbreakable token, so it has to be allowed
          to wrap -- left to itself it sets the width of the whole table */}
      <Typography.Text
        type='secondary'
        className={classNames('flex-1 min-w-0', { 'text-xs break-all': !masked })}
      >
        {masked ? '*********************' : url}
      </Typography.Text>
      <Button
        size='small'
        className='shrink-0'
        icon={<FontAwesomeIcon icon={masked ? faEye : faEyeSlash} />}
        onClick={() => setMasked(!masked)}
        type='text'
      />
      <Button
        size='small'
        className='shrink-0'
        icon={<FontAwesomeIcon icon={faClipboard} />}
        onClick={async () => {
          await navigator.clipboard.writeText(url);
          notification.success({ message: 'URL copied.', placement: 'bottomRight' });
        }}
        type='text'
      />
    </Flex>
  );
};

/**
 * A receiver's position in the full list is what identifies it to a save, since
 * the index AntD hands a column's render counts from the top of the current
 * page only. The URL comes from the config's status rather than its spec, so it
 * travels with the row rather than being looked up again per column.
 */
type WebhookReceiverRow = {
  config: WebhookReceiverConfig;
  index: number;
  url?: string;
};

type WebhookReceiversListViewProps = {
  webhookReceivers: WebhookReceiverConfig[];
  receiverDetails: WebhookReceiverDetails[];
  loading?: boolean;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
};

export const WebhookReceiversListView = ({
  webhookReceivers,
  receiverDetails,
  loading,
  onEdit,
  onDelete
}: WebhookReceiversListViewProps) => {
  const urls = new Map(receiverDetails.map((details) => [details.name, details.url]));

  return (
    <Table<WebhookReceiverRow>
      loading={loading}
      rowKey='index'
      dataSource={webhookReceivers.map((config, index) => ({
        config,
        index,
        url: urls.get(config.name)
      }))}
      pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
      size='small'
      scroll={{ x: 'max-content' }}
      locale={{
        emptyText: 'No receivers yet - nothing can notify Kargo of new artifacts.'
      }}
      columns={[
        {
          key: 'type',
          title: 'Receiver',
          render: (_, { config }) => <ReceiverCell config={config} />
        },
        {
          key: 'name',
          title: 'Name',
          render: (_, { config }) => <Typography.Text strong>{config.name}</Typography.Text>
        },
        {
          key: 'secret',
          title: 'Secret',
          render: (_, { config }) => (
            <Typography.Text code className='text-xs'>
              {receiverSecretName(config)}
            </Typography.Text>
          )
        },
        {
          key: 'url',
          title: 'URL',
          // a px width, not a percentage: with scroll.x the table sizes to its
          // content, and a percentage of that has nothing definite to resolve
          // against. Fixed, the layout holds still as the URL is revealed
          width: 360,
          render: (_, { url }) => <URLCell url={url} />
        },
        {
          key: 'actions',
          fixed: 'right',
          // the two buttons are the same on every row, so pinning the width
          // keeps the column from resizing as the rest of the table reflows
          width: 180,
          render: (_, { config, index }) => (
            <Flex gap={8} justify='end'>
              {/* the form writes a Secret reference into a receiver's type
                  block, so one with no type has nowhere to put it */}
              <Tooltip
                title={
                  isTypelessReceiver(config)
                    ? 'This receiver names no type, so it cannot be edited here. Give it one in the config YAML, or delete it and create a replacement.'
                    : undefined
                }
              >
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  onClick={() => onEdit(index)}
                  disabled={isTypelessReceiver(config)}
                  color='default'
                  variant='filled'
                  size='small'
                >
                  Edit
                </Button>
              </Tooltip>
              <Button
                icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                onClick={() => onDelete(index)}
                color='danger'
                variant='filled'
                size='small'
              >
                Delete
              </Button>
            </Flex>
          )
        }
      ]}
    />
  );
};
