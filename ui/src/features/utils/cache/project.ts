import { createConnectQueryKey, createProtobufSafeUpdater } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { listProjects } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Project } from '@ui/gen/v1alpha1/generated_pb';

export default {
  add: (projects: Project[]) => {
    queryClient.setQueriesData(
      {
        queryKey: createConnectQueryKey(listProjects)
          // IMPORTANT: createConnectQueryKey returns falsy elements for filters so lets use only static identifiers
          .slice(0, 2),
        exact: false
      },
      createProtobufSafeUpdater(listProjects, (prev) => {
        let newProjects = [...(prev?.projects || [])];

        if (projects?.length > 0) {
          newProjects = newProjects.concat(projects);
        }

        return {
          ...prev,
          total: newProjects.length,
          projects: newProjects
        };
      })
    );
  }
};
