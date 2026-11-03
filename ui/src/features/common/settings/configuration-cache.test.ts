import { MutationObserver, QueryClient } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';

import { getGetProjectConfigQueryKey } from '../../../gen/api/v2/core/core';
import { getGetClusterConfigQueryKey } from '../../../gen/api/v2/system/system';

import {
  configurationForUpdate,
  configurationMutationScope,
  updateConfigurationCache
} from './configuration-cache';

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

describe('serialized configuration saves', () => {
  it('keeps consecutive patches when a new informer read would still return the old spec', async () => {
    const client = new QueryClient();
    const key = getGetClusterConfigQueryKey();
    const initial = { ...saved, spec: {} };
    client.setQueryData(key, { data: initial, status: 200, headers: new Headers() });
    const fetch = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(new Response(JSON.stringify(initial)));
    try {
      await updateConfigurationCache(client, undefined, [{ updatedResourceManifest: saved }]);
      const next = await configurationForUpdate(client, undefined);
      const second = { ...next, spec: { ...next?.spec, promotionPolicies: [] } };
      await updateConfigurationCache(client, undefined, [{ updatedResourceManifest: second }]);
      expect(client.getQueryData<{ data: typeof saved }>(key)?.data.spec).toMatchObject(saved.spec);
      expect(fetch).not.toHaveBeenCalled();
    } finally {
      fetch.mockRestore();
      client.clear();
    }
  });

  it('serializes different editors and builds the queued save from the previous confirmed response', async () => {
    const client = new QueryClient();
    const project = {
      ...saved,
      kind: 'ProjectConfig',
      metadata: { name: 'owned', namespace: 'owned' },
      spec: {}
    };
    const key = getGetProjectConfigQueryKey('owned');
    client.setQueryData(key, { data: project, status: 200, headers: new Headers() });
    const started: number[] = [];
    let release!: () => void;
    const firstResponse = new Promise<void>((resolve) => {
      release = resolve;
    });
    const editor = (index: number) =>
      new MutationObserver(client, {
        scope: configurationMutationScope('owned'),
        mutationFn: async (patch: Record<string, unknown>) => {
          started.push(index);
          const current = await configurationForUpdate(client, 'owned');
          if (index === 1) await firstResponse;
          return { ...current, spec: { ...current?.spec, ...patch } };
        },
        onSuccess: (result) =>
          updateConfigurationCache(client, 'owned', [{ updatedResourceManifest: result }])
      });
    const first = editor(1);
    const second = editor(2);
    const firstSave = first.mutate({ stageLinks: saved.spec.stageLinks });
    await vi.waitFor(() => expect(started).toEqual([1]));
    const secondSave = second.mutate({ webhookReceivers: saved.spec.webhookReceivers });
    await vi.waitFor(() => expect(second.getCurrentResult().isPaused).toBe(true));
    expect(started).toEqual([1]);
    release();
    await Promise.all([firstSave, secondSave]);
    expect(started).toEqual([1, 2]);
    expect(client.getQueryData<{ data: typeof saved }>(key)?.data.spec).toEqual(saved.spec);
    expect(configurationMutationScope('other')).not.toEqual(configurationMutationScope('owned'));
    expect(configurationMutationScope(undefined)).not.toEqual(configurationMutationScope('owned'));
    client.clear();
  });
});
