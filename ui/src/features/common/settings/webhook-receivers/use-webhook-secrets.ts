import {
  useCreateProjectGenericCredentials,
  useCreateSystemGenericCredentials,
  useDeleteProjectGenericCredentials,
  useDeleteSystemGenericCredentials,
  useListProjectGenericCredentials,
  useListSystemGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';

/**
 * The Secrets a webhook receiver may authenticate against, and the two things
 * the form does with them. Which namespace they live in depends on the scope of
 * the config being edited -- the Project's own for a ProjectConfig, the system
 * resources namespace for a ClusterConfig -- so each scope supplies its own.
 */
export type WebhookSecrets = {
  /**
   * Names of the Secrets in scope, offered as suggestions. Listing them is not
   * essential: a name may always be typed, which is what keeps the form working
   * for an installation whose secret management is turned off.
   */
  names: string[];
  create: (name: string, data: Record<string, string>) => Promise<unknown>;
  remove: (name: string) => Promise<unknown>;
};

const namesOf = (secrets?: V1Secret[]) =>
  (secrets ?? []).map((secret) => secret.metadata?.name ?? '').filter(Boolean);

export const useProjectWebhookSecrets = (project: string): WebhookSecrets => {
  const listQuery = useListProjectGenericCredentials(project);
  const createMutation = useCreateProjectGenericCredentials();
  const deleteMutation = useDeleteProjectGenericCredentials();

  return {
    names: namesOf(listQuery.data?.data?.items),
    create: async (name, data) => {
      await createMutation.mutateAsync({ project, data: { name, data } });
      await listQuery.refetch();
    },
    remove: async (genericCredentials) => {
      await deleteMutation.mutateAsync({ project, genericCredentials });
      await listQuery.refetch();
    }
  };
};

export const useClusterWebhookSecrets = (): WebhookSecrets => {
  const listQuery = useListSystemGenericCredentials();
  const createMutation = useCreateSystemGenericCredentials();
  const deleteMutation = useDeleteSystemGenericCredentials();

  return {
    names: namesOf(listQuery.data?.data?.items),
    create: async (name, data) => {
      await createMutation.mutateAsync({ data: { name, data } });
      await listQuery.refetch();
    },
    remove: async (genericCredentials) => {
      await deleteMutation.mutateAsync({ genericCredentials });
      await listQuery.refetch();
    }
  };
};
