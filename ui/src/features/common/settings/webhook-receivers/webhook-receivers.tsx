import { faExternalLink, faPlus, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Typography } from 'antd';
import { ReactNode } from 'react';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { WebhookReceiverConfig, WebhookReceiverDetails } from '@ui/gen/api/v2/models';

import { WebhookSecrets } from './use-webhook-secrets';
import { receiverSecretName } from './webhook-receiver-form-utils';
import { NewWebhookSecret, WebhookReceiverModal } from './webhook-receiver-modal';
import { WebhookReceiversListView } from './webhook-receivers-list-view';

type WebhookReceiversProps = {
  webhookReceivers: WebhookReceiverConfig[];
  receiverDetails: WebhookReceiverDetails[];
  secrets: WebhookSecrets;
  loading?: boolean;
  /**
   * Whether the config failed to load for a reason other than not existing yet.
   * A save writes the whole spec back, so it must not run against a spec this
   * page never managed to read.
   */
  loadFailed?: boolean;
  onUpdate: (webhookReceivers: WebhookReceiverConfig[]) => Promise<unknown>;
  refresh: ReactNode;
};

export const WebhookReceivers = ({
  webhookReceivers,
  receiverDetails,
  secrets,
  loading,
  loadFailed,
  onUpdate,
  refresh
}: WebhookReceiversProps) => {
  const createModal = useModal();
  const editModal = useModal();
  const confirm = useConfirmModal();

  const names = webhookReceivers.map((receiver) => receiver.name);

  /**
   * Saves the receiver list, first creating the Secret it leans on when the
   * form asked for a new one. A Secret created for a config that then fails to
   * save is removed again, so a rejected save leaves nothing behind.
   */
  const save = async (next: WebhookReceiverConfig[], newSecret?: NewWebhookSecret) => {
    if (!newSecret) {
      return onUpdate(next);
    }

    await secrets.create(newSecret.name, newSecret.data);

    try {
      return await onUpdate(next);
    } catch (err) {
      await secrets.remove(newSecret.name);
      throw err;
    }
  };

  const onCreate = () =>
    createModal.show((p) => (
      <WebhookReceiverModal
        {...p}
        secrets={secrets}
        names={names}
        onSubmit={(receiver, newSecret) => save([...webhookReceivers, receiver], newSecret)}
      />
    ));

  const onEdit = (index: number) =>
    editModal.show((p) => (
      <WebhookReceiverModal
        {...p}
        editing
        secrets={secrets}
        names={names.filter((_, i) => i !== index)}
        webhookReceiver={webhookReceivers[index]}
        onSubmit={(receiver, newSecret) =>
          save(
            webhookReceivers.map((existing, i) => (i === index ? receiver : existing)),
            newSecret
          )
        }
      />
    ));

  const onDelete = (index: number) => {
    const receiver = webhookReceivers[index];

    confirm({
      title: 'Delete Webhook Receiver',
      content: (
        <Flex vertical gap={8}>
          <Typography.Text>
            Are you sure you want to delete the webhook receiver{' '}
            <Typography.Text strong>{receiver.name}</Typography.Text>? Its URL stops working, and
            whatever posts to it can no longer tell Kargo about new artifacts.
          </Typography.Text>
          {/* another receiver may lean on the same Secret, so deleting it here
              would be a guess -- name it instead and leave it to be cleaned up */}
          <Typography.Text type='secondary'>
            The Secret <Typography.Text code>{receiverSecretName(receiver)}</Typography.Text> is
            left in place.
          </Typography.Text>
        </Flex>
      ),
      onOk: () => onUpdate(webhookReceivers.filter((_, i) => i !== index))
    });
  };

  return (
    <Card
      title={
        <Space size={4}>
          Webhook Receivers
          <Popover
            content={
              <Flex vertical gap={6} className='max-w-xs text-xs'>
                <Typography.Text>
                  A receiver gives Kargo a URL that a registry or Git host can post to, so
                  Warehouses learn about new artifacts as they appear instead of on the next poll.
                </Typography.Text>
                <Typography.Text>
                  Each one authenticates requests against a Secret, and Kargo builds the URL from
                  that Secret - so the URL itself is sensitive, and rotating the Secret changes it.
                </Typography.Text>
                <Typography.Text>
                  Receivers refresh every Warehouse subscribed to the repository a request came
                  from. What happens next is up to those Warehouses.
                </Typography.Text>
                <Typography.Link
                  href='https://docs.kargo.io/user-guide/reference-docs/webhook-receivers'
                  target='_blank'
                  className='text-xs whitespace-nowrap'
                >
                  Learn more
                  <FontAwesomeIcon icon={faExternalLink} className='ml-1' size='xs' />
                </Typography.Link>
              </Flex>
            }
          >
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      type='inner'
      className='min-h-full'
      extra={
        <Space>
          {/* refreshing the config re-mints receiver URLs after a Secret
              rotation -- with no receivers there is nothing to re-mint */}
          {webhookReceivers.length > 0 && refresh}
          {/* the modal captures the receiver list as it opens -- until the
              config has loaded (or is known not to exist) that list is empty,
              and saving would replace everything already stored */}
          <Button
            icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
            onClick={onCreate}
            disabled={loading || loadFailed}
          >
            New Receiver
          </Button>
        </Space>
      }
    >
      <WebhookReceiversListView
        webhookReceivers={webhookReceivers}
        receiverDetails={receiverDetails}
        loading={loading}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </Card>
  );
};
