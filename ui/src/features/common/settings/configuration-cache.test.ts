import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import { getGetProjectConfigQueryKey } from '../../../gen/api/v2/core/core';
import { getGetClusterConfigQueryKey } from '../../../gen/api/v2/system/system';

import { updateConfigurationCache } from './configuration-cache';

const saved = {
  kind: 'ClusterConfig',
  metadata: { name: 'cluster', resourceVersion: '2' },
  spec: {
    stageLinks: [{ title: 'link', url: 'https://example.invalid' }],
    webhookReceivers: [{ name: 'keep', gitea: { secretRef: { name: 'owned' } } }]
  }
};

describe('confirmed configuration cache', () => {
  it('retains a saved link and unrelated receiver when an older read finishes late', async () => {
    const client = new QueryClient();
    const key = getGetClusterConfigQueryKey();
    let finish!: (value: unknown) => void;
    const pending = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await updateConfigurationCache(client, undefined, [{ updatedResourceManifest: saved }]);
    finish({ data: { ...saved, spec: {} }, status: 200 });
    await pending;
    expect(client.getQueryData<{ data: typeof saved }>(key)?.data).toEqual(saved);
    client.clear();
  });

  it('handles creation in the selected project without changing other scopes', async () => {
    const client = new QueryClient();
    const project = {
      ...saved,
      kind: 'ProjectConfig',
      metadata: { name: 'owned', namespace: 'owned' }
    };
    client.setQueryData(getGetClusterConfigQueryKey(), { data: saved });
    await updateConfigurationCache(client, 'owned', [{ createdResourceManifest: project }]);
    expect(
      client.getQueryData<{ data: typeof project }>(getGetProjectConfigQueryKey('owned'))?.data
    ).toEqual(project);
    expect(client.getQueryData(getGetProjectConfigQueryKey('other'))).toBeUndefined();
    expect(client.getQueryData(getGetClusterConfigQueryKey())).toEqual({ data: saved });
    client.clear();
  });

  it('does not apply failed, foreign or malformed mutation results', async () => {
    const client = new QueryClient();
    await updateConfigurationCache(client, undefined, [
      { error: 'denied', updatedResourceManifest: saved },
      { updatedResourceManifest: { ...saved, kind: 'ProjectConfig' } },
      { updatedResourceManifest: { ...saved, metadata: { name: 'other' } } },
      { updatedResourceManifest: { kind: 'ClusterConfig' } }
    ]);
    expect(client.getQueryData(getGetClusterConfigQueryKey())).toBeUndefined();
    client.clear();
  });

  it('retains policy edits and removals together with unrelated links and receivers', async () => {
    const client = new QueryClient();
    const key = getGetProjectConfigQueryKey('owned');
    const project = {
      ...saved,
      kind: 'ProjectConfig',
      metadata: { name: 'owned', namespace: 'owned' },
      spec: {
        ...saved.spec,
        promotionPolicies: [{ stageSelector: { name: 'dev' }, autoPromotionEnabled: true }]
      }
    };
    await updateConfigurationCache(client, 'owned', [{ updatedResourceManifest: project }]);
    expect(client.getQueryData<{ data: typeof project }>(key)?.data).toEqual(project);
    const removed = { ...project, spec: { ...project.spec, promotionPolicies: [] } };
    await updateConfigurationCache(client, 'owned', [{ updatedResourceManifest: removed }]);
    expect(client.getQueryData<{ data: typeof removed }>(key)?.data).toEqual(removed);
    client.clear();
  });
});
