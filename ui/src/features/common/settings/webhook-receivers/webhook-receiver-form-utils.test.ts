import { describe, expect, it } from 'vitest';

import { WebhookReceiverConfig } from '@ui/gen/api/v2/models';

import {
  emptySecretData,
  formValuesFromWebhookReceiver,
  isTypelessReceiver,
  receiverSecretName,
  receiverTypeKey,
  webhookReceiverFormSchema,
  WebhookReceiverFormValues,
  webhookReceiverFromFormValues
} from './webhook-receiver-form-utils';

const githubReceiver: WebhookReceiverConfig = {
  name: 'my-receiver',
  github: { secretRef: { name: 'my-secret' } }
};

const formValues = (
  overrides: Partial<WebhookReceiverFormValues> = {}
): WebhookReceiverFormValues => ({
  type: 'github',
  name: 'my-receiver',
  secretMode: 'existing',
  secretName: 'my-secret',
  secretData: {},
  virtualRepoName: '',
  ...overrides
});

describe('receiverTypeKey', () => {
  it('finds the configured type', () => {
    expect(receiverTypeKey(githubReceiver)).toBe('github');
  });

  it('prefers a known type over anything else alongside it', () => {
    const config = { name: 'r', somethingNew: {}, gitlab: {} } as WebhookReceiverConfig;

    expect(receiverTypeKey(config)).toBe('gitlab');
  });

  it('falls back to a type it does not know', () => {
    const config = { name: 'r', harbor: {} } as WebhookReceiverConfig;

    expect(receiverTypeKey(config)).toBe('harbor');
  });

  it('is empty for a receiver with no type at all', () => {
    expect(receiverTypeKey({ name: 'r' })).toBe('');
  });
});

describe('isTypelessReceiver', () => {
  it('spots a receiver with nothing but a name', () => {
    expect(isTypelessReceiver({ name: 'r' })).toBe(true);
  });

  it('accepts a receiver of a type it does not know', () => {
    expect(isTypelessReceiver({ name: 'r', harbor: {} } as WebhookReceiverConfig)).toBe(false);
  });

  it('accepts an ordinary receiver', () => {
    expect(isTypelessReceiver(githubReceiver)).toBe(false);
  });
});

describe('receiverSecretName', () => {
  it('reads the reference out of the type block', () => {
    expect(receiverSecretName(githubReceiver)).toBe('my-secret');
  });

  it('is empty when there is no reference', () => {
    expect(receiverSecretName({ name: 'r' })).toBe('');
  });
});

describe('emptySecretData', () => {
  it('covers every key the type expects', () => {
    expect(emptySecretData('gitlab')).toEqual({ 'secret-token': '' });
  });

  it('is empty for a type it does not know', () => {
    expect(emptySecretData('harbor')).toEqual({});
  });
});

describe('formValuesFromWebhookReceiver', () => {
  it('starts a new receiver on the first type, creating a Secret', () => {
    const values = formValuesFromWebhookReceiver();

    expect(values.type).toBe('azure');
    expect(values.name).toBe('');
    expect(values.secretMode).toBe('new');
    expect(values.secretName).toBe('');
  });

  it('points an existing receiver at the Secret it already has', () => {
    expect(formValuesFromWebhookReceiver(githubReceiver)).toEqual(
      formValues({ secretData: { secret: '' } })
    );
  });

  it('reads an Artifactory receiver virtual repository', () => {
    const config: WebhookReceiverConfig = {
      name: 'r',
      artifactory: { secretRef: { name: 's' }, virtualRepoName: 'proj-virtual' }
    };

    expect(formValuesFromWebhookReceiver(config).virtualRepoName).toBe('proj-virtual');
  });
});

describe('webhookReceiverFromFormValues', () => {
  it('nests the Secret reference under the chosen type', () => {
    expect(webhookReceiverFromFormValues(formValues())).toEqual(githubReceiver);
  });

  it('keeps the type of the receiver being edited, whatever the form holds', () => {
    const edited = webhookReceiverFromFormValues(
      formValues({ type: 'gitlab', name: 'renamed', secretName: 'other-secret' }),
      githubReceiver
    );

    expect(edited).toEqual({ name: 'renamed', github: { secretRef: { name: 'other-secret' } } });
  });

  it('carries over configuration the form never showed', () => {
    const generic = {
      name: 'r',
      generic: { secretRef: { name: 's' }, actions: [{ action: 'Refresh' }] }
    } as WebhookReceiverConfig;

    const edited = webhookReceiverFromFormValues(
      formValues({ type: 'generic', name: 'r', secretName: 'next' }),
      generic
    );

    expect(edited).toEqual({
      name: 'r',
      generic: { secretRef: { name: 'next' }, actions: [{ action: 'Refresh' }] }
    });
  });

  it('sets an Artifactory virtual repository', () => {
    const created = webhookReceiverFromFormValues(
      formValues({ type: 'artifactory', virtualRepoName: 'proj-virtual' })
    );

    expect(created).toEqual({
      name: 'my-receiver',
      artifactory: { secretRef: { name: 'my-secret' }, virtualRepoName: 'proj-virtual' }
    });
  });

  it('drops an Artifactory virtual repository that was cleared', () => {
    const config: WebhookReceiverConfig = {
      name: 'my-receiver',
      artifactory: { secretRef: { name: 'my-secret' }, virtualRepoName: 'proj-virtual' }
    };

    const edited = webhookReceiverFromFormValues(
      formValues({ type: 'artifactory', virtualRepoName: '' }),
      config
    );

    expect(edited).toEqual({
      name: 'my-receiver',
      artifactory: { secretRef: { name: 'my-secret' } }
    });
  });

  it('leaves the virtual repository alone for every other type', () => {
    const created = webhookReceiverFromFormValues(formValues({ virtualRepoName: 'ignored' }));

    expect(created).toEqual(githubReceiver);
  });
});

describe('webhookReceiverFormSchema', () => {
  const errorPaths = (values: WebhookReceiverFormValues) => {
    const result = webhookReceiverFormSchema.safeParse(values);

    return result.success ? [] : result.error.issues.map((issue) => issue.path.join('.'));
  };

  it('accepts a receiver pointed at an existing Secret', () => {
    expect(errorPaths(formValues())).toEqual([]);
  });

  it('rejects names that are not DNS subdomains', () => {
    expect(errorPaths(formValues({ name: 'My Receiver' }))).toEqual(['name']);
    expect(errorPaths(formValues({ secretName: 'My Secret' }))).toEqual(['secretName']);
  });

  it('requires every key a new Secret is expected to carry', () => {
    expect(errorPaths(formValues({ secretMode: 'new', secretData: {} }))).toEqual([
      'secretData.secret'
    ]);
  });

  it('does not accept whitespace in place of a Secret value', () => {
    const values = formValues({ secretMode: 'new', secretData: { secret: '  ' } });

    expect(errorPaths(values)).toEqual(['secretData.secret']);
  });

  it('reports a hyphenated Secret key at the path the form binds to', () => {
    const values = formValues({ type: 'gitlab', secretMode: 'new', secretData: {} });

    expect(errorPaths(values)).toEqual(['secretData.secret-token']);
  });

  it('accepts a filled-in new Secret', () => {
    const values = formValues({ secretMode: 'new', secretData: { secret: 'shhh' } });

    expect(errorPaths(values)).toEqual([]);
  });

  it('asks for no Secret keys for a type it does not know', () => {
    const values = formValues({ type: 'harbor', secretMode: 'new', secretData: {} });

    expect(errorPaths(values)).toEqual([]);
  });
});
