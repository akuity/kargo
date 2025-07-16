import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Form, Input, Modal, Select } from 'antd';
import TextArea from 'antd/es/input/TextArea';
import { useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { webhookReceivers } from '@ui/features/project/settings/views/project-config/webhook/webhook-receivers';

import { useCreateWebhookMutation } from './use-create-webhook-mutation';

type CreateWebhookModalProps = ModalComponentProps & {
  clusterConfigYAML: string;
  project: string;
};

const secretFormSchema = z.object({
  webhookReceiverName: z.string().nonempty(),
  name: z.string().nonempty(),
  data: z.record(z.string(), z.string().nonempty())
});

export const CreateWebhookModal = (props: CreateWebhookModalProps) => {
  const [webhookReceiver, setWebhookReceiver] = useState(webhookReceivers[0].key);

  const webhookReceiverExpand = useMemo(
    () => webhookReceivers.find((r) => r.key === webhookReceiver)!,
    [webhookReceiver]
  );

  const secretForm = useForm({
    defaultValues: {
      webhookReceiverName: '',
      name: '',
      data: {}
    },
    resolver: zodResolver(secretFormSchema)
  });

  const createWebhookMutation = useCreateWebhookMutation({
    onSuccess: () => props.hide()
  });

  const handleSubmit = secretForm.handleSubmit((data) => {
    createWebhookMutation.mutate({
      clusterConfigYAML: props.clusterConfigYAML,
      secret: {
        name: data.name,
        data: data.data
      },
      webhookReceiver,
      webhookReceiverName: data.webhookReceiverName
    });
  });

  return (
    <Modal
      open={props.visible}
      onCancel={props.hide}
      okText='Add'
      onOk={handleSubmit}
      okButtonProps={{ loading: createWebhookMutation.isPending }}
    >
      <Form layout='vertical'>
        <Form.Item label='Receiver'>
          <Select
            value={webhookReceiver}
            options={webhookReceivers.map((r) => ({
              value: r.key,
              label: (
                <>
                  {r.icon && <FontAwesomeIcon icon={r.icon} className='mr-2' />}
                  {r.label}
                </>
              )
            }))}
            onChange={(value) => {
              setWebhookReceiver(value);
              secretForm.reset();
            }}
          />
        </Form.Item>
      </Form>

      <FieldContainer control={secretForm.control} name='webhookReceiverName' label='Name'>
        {({ field }) => (
          <Input
            placeholder='my-webhook-receiver'
            value={field.value}
            onChange={(e) => field.onChange(e.target.value)}
          />
        )}
      </FieldContainer>

      <b>Secret</b>

      <FieldContainer className='mt-2' control={secretForm.control} label='Secret Name' name='name'>
        {({ field }) => (
          <Input
            placeholder={`my-${webhookReceiver}-secret`}
            value={field.value}
            onChange={field.onChange}
          />
        )}
      </FieldContainer>

      <FieldContainer control={secretForm.control} name='data'>
        {({ field }) => {
          const value = field.value;
          return webhookReceiverExpand.secrets.map((secret) => {
            return (
              <Form key={secret.dataKey} layout='vertical'>
                <Form.Item label={secret.dataKey}>
                  <TextArea
                    rows={1}
                    value={value[secret.dataKey]}
                    onChange={(nextValue) =>
                      field.onChange({
                        ...value,
                        [secret.dataKey]: nextValue.target.value
                      })
                    }
                  />
                  {secret.description && (
                    <div className='text-xs text-gray-500 mt-2'>{secret.description}</div>
                  )}
                </Form.Item>
              </Form>
            );
          });
        }}
      </FieldContainer>
    </Modal>
  );
};
