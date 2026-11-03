import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import {
  getGetAnalysisTemplateQueryKey,
  getListAnalysisTemplatesQueryKey,
  listAnalysisTemplatesResponse
} from '../../../../../gen/api/v2/verifications/verifications';

import { removeAnalysisTemplateFromCache, updateAnalysisTemplateCache } from './template-cache';

const key = getListAnalysisTemplatesQueryKey('owned-project');
const response = (names: string[]): listAnalysisTemplatesResponse => ({
  status: 200,
  headers: new Headers(),
  data: { items: names.map((name) => ({ metadata: { name }, spec: {} })) }
});

describe('AnalysisTemplate deletion cache', () => {
  it('does not resurrect a confirmed deletion when an older list completes', async () => {
    const client = new QueryClient();
    client.setQueryData(key, response(['deleted', 'untouched']));
    let finish!: (value: listAnalysisTemplatesResponse) => void;
    const pending = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise<listAnalysisTemplatesResponse>((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await removeAnalysisTemplateFromCache(client, 'owned-project', 'deleted');
    finish(response(['deleted', 'untouched']));
    await pending;
    expect(
      client
        .getQueryData<listAnalysisTemplatesResponse>(key)
        ?.data.items?.map((item) => item.metadata?.name)
    ).toEqual(['untouched']);
    client.clear();
  });

  it('preserves the same template name in a different project and does not synthesize missing data', async () => {
    const client = new QueryClient();
    const foreignKey = getListAnalysisTemplatesQueryKey('other-project');
    const foreign = response(['deleted']);
    client.setQueryData(foreignKey, foreign);
    await removeAnalysisTemplateFromCache(client, 'owned-project', 'deleted');
    expect(client.getQueryData(key)).toBeUndefined();
    expect(client.getQueryData(foreignKey)).toEqual(foreign);
    client.clear();
  });
});

describe('confirmed AnalysisTemplate saves', () => {
  const saved = {
    kind: 'AnalysisTemplate',
    metadata: { name: 'new', namespace: 'owned-project' },
    spec: { args: [{ name: 'message', value: 'saved' }] }
  };
  it('keeps a created row and detail after an older empty list finishes', async () => {
    const client = new QueryClient();
    let finish!: (value: listAnalysisTemplatesResponse) => void;
    const pending = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise<listAnalysisTemplatesResponse>((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await updateAnalysisTemplateCache(client, 'owned-project', [
      { createdResourceManifest: saved }
    ]);
    finish(response([]));
    await pending;
    expect(client.getQueryData<listAnalysisTemplatesResponse>(key)?.data.items).toEqual([saved]);
    expect(
      client.getQueryData<{ data: unknown }>(getGetAnalysisTemplateQueryKey('owned-project', 'new'))
        ?.data
    ).toEqual(saved);
    client.clear();
  });
  it('updates only the saved template and preserves unrelated rows and projects', async () => {
    const client = new QueryClient();
    client.setQueryData(key, response(['new', 'untouched']));
    const otherKey = getListAnalysisTemplatesQueryKey('other-project');
    client.setQueryData(otherKey, response(['new']));
    await updateAnalysisTemplateCache(client, 'owned-project', [
      { updatedResourceManifest: saved }
    ]);
    expect(client.getQueryData<listAnalysisTemplatesResponse>(key)?.data.items).toEqual([
      saved,
      { metadata: { name: 'untouched' }, spec: {} }
    ]);
    expect(client.getQueryData(otherKey)).toEqual(response(['new']));
    client.clear();
  });
  it('ignores failed writes, wrong resource kinds and foreign namespaces', async () => {
    const client = new QueryClient();
    await updateAnalysisTemplateCache(client, 'owned-project', [
      { error: 'denied', updatedResourceManifest: saved },
      { createdResourceManifest: { ...saved, kind: 'ClusterAnalysisTemplate' } },
      {
        createdResourceManifest: { ...saved, metadata: { ...saved.metadata, namespace: 'foreign' } }
      },
      { createdResourceManifest: { kind: 'AnalysisTemplate' } }
    ]);
    expect(client.getQueryData(key)).toBeUndefined();
    client.clear();
  });
});
