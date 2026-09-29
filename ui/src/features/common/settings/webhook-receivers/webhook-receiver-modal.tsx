import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { AutoComplete, Input, Modal, Segmented, Select, Tooltip, Typography } from 'antd';
import TextArea from 'antd/es/input/TextArea';
import { useState } from 'react';
import { useForm } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { WebhookReceiverConfig } from '@ui/gen/api/v2/models';

import { webhookReceivers } from './receiver-types';
import { WebhookSecrets } from './use-webhook-secrets';
import {
  artifactoryKey,
  emptySecretData,
  formValuesFromWebhookReceiver,
  receiverType,
  receiverTypeLabel,
  webhookReceiverFormSchema,
  WebhookReceiverFormValues,
  webhookReceiverFromFormValues
} from './webhook-receiver-form-utils';

/** A Secret to create alongside the receiver that will reference it. */
export type NewWebhookSecret = { name: string; data: Record<string, string> };

const secretNamespace = (scope: 'project' | 'cluster') =>
  scope === 'cluster' ? "Kargo's system resources namespace" : "the Project's namespace";

type WebhookReceiverModalProps = ModalComponentProps & {
  scope: 'project' | 'cluster';
  secrets: WebhookSecrets;
  /** Names already taken by the other receivers on the same config. */
  names: string[];
  editing?: boolean;
  webhookReceiver?: WebhookReceiverConfig;
  onSubmit: (
    webhookReceiver: WebhookReceiverConfig,
    newSecret?: NewWebhookSecret
  ) => Promise<unknown>;
};

export const WebhookReceiverModal = ({
  visible,
  hide,
  scope,
  secrets,
  names,
  editing,
  webhookReceiver,
  onSubmit
}: WebhookReceiverModalProps) => {
  const form = useForm<WebhookReceiverFormValues>({
    defaultValues: formValuesFromWebhookReceiver(webhookReceiver),
    resolver: zodResolver(webhookReceiverFormSchema)
  });

  const [submitting, setSubmitting] = useState(false);

  const type = form.watch('type');
  const name = form.watch('name');
  const secretMode = form.watch('secretMode');

  const nameTaken = names.includes(name.trim());

  // a receiver whose type this form does not know cannot have a Secret written
  // for it -- which keys that Secret needs is exactly what is not known. It is
  // still the type the receiver has, so the select offers that and nothing else
  const knownType = !!receiverType(type);

  const typeOptions = (knownType ? webhookReceivers : []).map((r) => ({
    value: r.key,
    label: (
      <>
        {r.icon && <FontAwesomeIcon icon={r.icon} className='mr-2' />}
        {r.label}
      </>
    )
  }));

  if (!knownType) {
    typeOptions.push({ value: type, label: <>{receiverTypeLabel(type)}</> });
  }

  const handleSubmit = form.handleSubmit(async (values) => {
    setSubmitting(true);
    try {
      await onSubmit(
        webhookReceiverFromFormValues(values, webhookReceiver),
        values.secretMode === 'new'
          ? { name: values.secretName, data: values.secretData }
          : undefined
      );
      hide();
    } catch {
      // the failure has already been reported by the mutation layer; all that
      // is left to do here is keep the form open with the values still in it
    } finally {
      setSubmitting(false);
    }
  });

  return (
    <Modal
      open={visible}
      onCancel={hide}
      width={680}
      title={editing ? 'Edit Webhook Receiver' : 'New Webhook Receiver'}
      okText={editing ? 'Save changes' : 'Create'}
      onOk={handleSubmit}
      okButtonProps={{ loading: submitting, disabled: nameTaken }}
    >
      <FieldContainer control={form.control} name='type' label='Receiver' required>
        {({ field }) => (
          <Tooltip
            title={
              editing
                ? 'A receiver keeps the type it was created with. Delete it and create a replacement to switch.'
                : undefined
            }
          >
            <Select
              value={field.value}
              onChange={(value) => {
                field.onChange(value);
                // the keys a Secret needs are the ones its own type asks for
                form.setValue('secretData', emptySecretData(value));
              }}
              disabled={editing}
              className='w-full'
              options={typeOptions}
            />
          </Tooltip>
        )}
      </FieldContainer>

      <FieldContainer control={form.control} name='name' label='Name' required>
        {({ field }) => (
          <>
            <Input {...field} placeholder='my-webhook-receiver' />
            {nameTaken && (
              <Typography.Text type='danger' className='text-xs'>
                Another receiver on this config already uses that name
              </Typography.Text>
            )}
          </>
        )}
      </FieldContainer>

      {type === artifactoryKey && (
        <FieldContainer
          control={form.control}
          name='virtualRepoName'
          label='Virtual repository'
          description='Name of the Artifactory virtual repository this receiver serves. Leave empty unless Warehouses subscribe to repositories through a virtual one.'
        >
          {({ field }) => <Input {...field} placeholder='proj-virtual' />}
        </FieldContainer>
      )}

      <Typography.Text strong>Secret</Typography.Text>

      <FieldContainer
        className='mt-2'
        control={form.control}
        name='secretMode'
        description={`Kargo authenticates inbound requests against a Secret in ${secretNamespace(scope)}, and builds the receiver's URL from it.`}
        formItemClassName='!mb-3'
      >
        {({ field }) => (
          <Segmented
            value={field.value}
            onChange={field.onChange}
            options={[
              { value: 'existing', label: 'Use an existing Secret' },
              { value: 'new', label: 'Create a new Secret', disabled: !knownType }
            ]}
          />
        )}
      </FieldContainer>

      <FieldContainer control={form.control} name='secretName' label='Secret name' required>
        {({ field }) =>
          secretMode === 'existing' ? (
            <AutoComplete
              className='w-full'
              value={field.value}
              onChange={field.onChange}
              placeholder='my-webhook-secret'
              options={secrets.names.map((value) => ({ value }))}
              filterOption={(input, option) =>
                (option?.value ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          ) : (
            <Input {...field} placeholder={`my-${type}-secret`} />
          )
        }
      </FieldContainer>

      {secretMode === 'new' &&
        receiverType(type)?.secrets.map((secret) => (
          <FieldContainer
            key={secret.dataKey}
            control={form.control}
            name={`secretData.${secret.dataKey}`}
            label={secret.dataKey}
            required
          >
            {({ field }) => (
              <>
                <TextArea {...field} rows={1} />
                {secret.description && (
                  <div className='text-xs text-gray-500 dark:text-neutral-400 mt-2'>
                    {secret.description}
                  </div>
                )}
              </>
            )}
          </FieldContainer>
        ))}
    </Modal>
  );
};
