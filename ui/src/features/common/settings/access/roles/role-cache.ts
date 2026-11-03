import { QueryClient } from '@tanstack/react-query';

import { RbacRole } from '@ui/gen/api/v2/models';
import { getListProjectRolesQueryKey, listProjectRolesResponse } from '@ui/gen/api/v2/rbac/rbac';

// Only call after the role endpoint confirms the mutation. System roles stay read-only.
export const updateProjectRoleCache = async (
  client: QueryClient,
  project: string,
  name: string,
  saved?: RbacRole
) => {
  if (
    !project ||
    !name ||
    (saved && (saved.metadata?.name !== name || saved.metadata.namespace !== project))
  )
    return;
  const key = getListProjectRolesQueryKey(project);
  await client.cancelQueries({ queryKey: key, exact: true });
  client.setQueryData<listProjectRolesResponse>(key, (old) => {
    const roles: RbacRole[] = Array.isArray(old?.data) ? old.data : [];
    const data = roles.filter((role) => role.metadata?.name !== name);
    if (saved) data.push(saved);
    data.sort((a, b) => (a.metadata?.name || '').localeCompare(b.metadata?.name || ''));
    return { ...(old || { status: 200 as const, headers: new Headers() }), data };
  });
};
