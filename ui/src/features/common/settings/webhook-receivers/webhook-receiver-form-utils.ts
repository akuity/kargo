import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { V1LocalObjectReference, WebhookReceiverConfig } from '@ui/gen/api/v2/models';
import { validatorMessages, zodValidators } from '@ui/utils/validators';

import { webhookReceivers } from './receiver-types';

/**
 * The catalog entry for a receiver type, or `undefined` for a type this form
 * does not know -- the API grows new receiver types faster than the catalog in
 * receiver-types does, and a config may already hold one of them.
 */
export const receiverType = (key: string) => webhookReceivers.find((type) => type.key === key);

/** A receiver type's display name, falling back to its raw API key. */
export const receiverTypeLabel = (key: string) => receiverType(key)?.label ?? key;

export const artifactoryKey = 'artifactory';

/**
 * A receiver's type-specific configuration. The generated
 * `WebhookReceiverConfig` spells every type out as its own optional field, but
 * exactly one is ever set and they all start from a Secret reference, so the
 * form treats them uniformly and leaves the rest of each block alone.
 */
type ReceiverBlock = {
  secretRef?: V1LocalObjectReference;
  [field: string]: unknown;
};

const blocksOf = (config: WebhookReceiverConfig) =>
  config as unknown as Record<string, ReceiverBlock | undefined>;

/**
 * Which of the API's receiver types a receiver is configured for. A known type
 * wins over an unknown one so that a config carrying something unexpected at
 * the top level still resolves to the type it actually describes.
 */
export const receiverTypeKey = (config: WebhookReceiverConfig): string => {
  const keys = Object.keys(config).filter((key) => key !== 'name');

  return keys.find((key) => !!receiverType(key)) ?? keys[0] ?? '';
};

/** The name of the Secret a receiver authenticates its requests with. */
export const receiverSecretName = (config: WebhookReceiverConfig): string =>
  blocksOf(config)[receiverTypeKey(config)]?.secretRef?.name ?? '';

/**
 * A receiver carrying no type block at all. Nothing rejects one -- neither the
 * CRD nor the validating webhook requires a type -- but the form has nowhere to
 * put a Secret reference, so editing it would mean inventing the type it is
 * missing. Such a receiver is left to the YAML editor.
 */
export const isTypelessReceiver = (config: WebhookReceiverConfig) => !receiverTypeKey(config);

const nameSchema = (subject: string) =>
  zodValidators.requiredString
    .max(253)
    .regex(dnsRegex, `${subject} must be a valid DNS subdomain.`);

export const webhookReceiverFormSchema = z
  .object({
    type: zodValidators.requiredString,
    name: nameSchema('Name'),
    /** Whether the receiver points at a Secret that exists or one to create. */
    secretMode: z.enum(['existing', 'new']),
    secretName: nameSchema('Secret name'),
    secretData: z.record(z.string(), z.string()),
    virtualRepoName: z.string()
  })
  .superRefine((values, ctx) => {
    if (values.secretMode !== 'new') {
      return;
    }

    // which keys a Secret must carry depends on the receiver type, so the
    // requirement cannot be expressed on the field itself
    for (const { dataKey } of receiverType(values.type)?.secrets ?? []) {
      if (!values.secretData[dataKey]?.trim()) {
        ctx.addIssue({
          code: 'custom',
          message: validatorMessages.required,
          path: ['secretData', dataKey]
        });
      }
    }
  });

export type WebhookReceiverFormValues = z.infer<typeof webhookReceiverFormSchema>;

/**
 * A blank value for every key a receiver type's Secret is expected to carry, so
 * that the fields standing for them start out controlled.
 */
export const emptySecretData = (type: string): Record<string, string> =>
  Object.fromEntries((receiverType(type)?.secrets ?? []).map(({ dataKey }) => [dataKey, '']));

export const formValuesFromWebhookReceiver = (
  config?: WebhookReceiverConfig
): WebhookReceiverFormValues => {
  const type = config ? receiverTypeKey(config) : webhookReceivers[0].key;
  const block = config ? blocksOf(config)[type] : undefined;

  return {
    type,
    name: config?.name ?? '',
    // an existing receiver already has a Secret; a new one starts out needing one
    secretMode: config ? 'existing' : 'new',
    secretName: block?.secretRef?.name ?? '',
    secretData: emptySecretData(type),
    virtualRepoName: typeof block?.virtualRepoName === 'string' ? block.virtualRepoName : ''
  };
};

/**
 * @param existing The receiver being edited, if this is an edit rather than a
 * create. Its type is never editable, and everything in its configuration the
 * form does not show -- a generic receiver's `actions`, say -- is carried over
 * untouched rather than dropped.
 */
export const webhookReceiverFromFormValues = (
  values: WebhookReceiverFormValues,
  existing?: WebhookReceiverConfig
): WebhookReceiverConfig => {
  const type = existing ? receiverTypeKey(existing) : values.type;

  const block: ReceiverBlock = {
    ...(existing ? blocksOf(existing)[type] : undefined),
    secretRef: { name: values.secretName }
  };

  if (type === artifactoryKey) {
    if (values.virtualRepoName) {
      block.virtualRepoName = values.virtualRepoName;
    } else {
      delete block.virtualRepoName;
    }
  }

  // the receiver type is a computed key, which no amount of typing will narrow
  // to the one optional field of WebhookReceiverConfig it stands for
  return { name: values.name, [type]: block } as WebhookReceiverConfig;
};
