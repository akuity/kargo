import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import { RbacRole } from '../../../../../gen/api/v2/models';
import {
  getListProjectRolesQueryKey,
  getListSystemRolesQueryKey
} from '../../../../../gen/api/v2/rbac/rbac';

import { updateProjectRoleCache } from './role-cache';

const role: RbacRole = {
  metadata: { name: 'reader', namespace: 'owned' },
  claims: [{ name: 'groups', values: ['readers'] }]
};
describe('confirmed project role cache', () => {
  it('keeps a created role after an older list finishes and applies update/delete without changing system roles', async () => {
    const client = new QueryClient();
    const key = getListProjectRolesQueryKey('owned');
    const systemKey = getListSystemRolesQueryKey();
    client.setQueryData(systemKey, { data: [role] });
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
    await updateProjectRoleCache(client, 'owned', 'reader', role);
    finish({ data: [] });
    await pending;
    expect(client.getQueryData<{ data: RbacRole[] }>(key)?.data).toEqual([role]);
    const edited = { ...role, claims: [{ name: 'groups', values: ['edited'] }] };
    await updateProjectRoleCache(client, 'owned', 'reader', edited);
    expect(client.getQueryData<{ data: RbacRole[] }>(key)?.data).toEqual([edited]);
    await updateProjectRoleCache(client, 'owned', 'reader');
    expect(client.getQueryData<{ data: RbacRole[] }>(key)?.data).toEqual([]);
    expect(client.getQueryData(systemKey)).toEqual({ data: [role] });
    client.clear();
  });
  it('does not apply a response from another project or another name', async () => {
    const client = new QueryClient();
    await updateProjectRoleCache(client, 'other', 'reader', role);
    await updateProjectRoleCache(client, 'owned', 'different', role);
    expect(client.getQueryData(getListProjectRolesQueryKey('other'))).toBeUndefined();
    expect(client.getQueryData(getListProjectRolesQueryKey('owned'))).toBeUndefined();
    client.clear();
  });
});
