import { QueryClient } from '@tanstack/react-query';

import { CreateOrUpdateResourceResult, RolloutsAnalysisTemplate } from '@ui/gen/api/v2/models';
import {
  getGetAnalysisTemplateQueryKey,
  getListAnalysisTemplatesQueryKey,
  getListAnalysisTemplatesQueryOptions,
  listAnalysisTemplatesResponse
} from '@ui/gen/api/v2/verifications/verifications';

export const updateAnalysisTemplateCache = async (
  client: QueryClient,
  project: string,
  results: CreateOrUpdateResourceResult[] = []
) => {
  const templates = results.flatMap((result) => {
    const manifest = result.updatedResourceManifest || result.createdResourceManifest;
    const metadata = manifest?.metadata as { name?: string; namespace?: string } | undefined;
    if (
      result.error ||
      manifest?.kind !== 'AnalysisTemplate' ||
      typeof metadata?.name !== 'string' ||
      metadata.namespace !== project
    )
      return [];
    return [manifest as unknown as RolloutsAnalysisTemplate];
  });
  if (!templates.length) return;
  const key = getListAnalysisTemplatesQueryKey(project);
  // If the first list is still loading, finish that read before merging the
  // confirmed save. Otherwise replacing it with one row hides existing templates.
  if (!client.getQueryData(key)) {
    await client.fetchQuery(getListAnalysisTemplatesQueryOptions(project));
  }
  await client.cancelQueries({ queryKey: key, exact: true });
  for (const template of templates) {
    const detailKey = getGetAnalysisTemplateQueryKey(project, template.metadata!.name!);
    await client.cancelQueries({ queryKey: detailKey, exact: true });
    client.setQueryData(detailKey, { data: template, status: 200, headers: new Headers() });
  }
  client.setQueryData<listAnalysisTemplatesResponse>(key, (old) => {
    const names = new Set(templates.map((template) => template.metadata?.name));
    const items = [
      ...(old?.data.items || []).filter((item) => !names.has(item.metadata?.name)),
      ...templates
    ];
    items.sort((a, b) => (a.metadata?.name || '').localeCompare(b.metadata?.name || ''));
    return {
      ...(old || { status: 200 as const, headers: new Headers() }),
      data: { ...old?.data, items }
    };
  });
};

// Run only after the delete endpoint confirms success. A refetch can still see
// the deleted object in the informer cache and otherwise leave it visible.
export const removeAnalysisTemplateFromCache = async (
  client: QueryClient,
  project: string,
  name: string
) => {
  const queryKey = getListAnalysisTemplatesQueryKey(project);
  await client.cancelQueries({ queryKey, exact: true });
  client.setQueryData<listAnalysisTemplatesResponse>(
    queryKey,
    (old) =>
      old && {
        ...old,
        data: { ...old.data, items: old.data.items?.filter((item) => item.metadata?.name !== name) }
      }
  );
};
