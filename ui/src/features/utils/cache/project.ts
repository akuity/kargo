import { create } from '@bufbuild/protobuf';
import { createConnectQueryKey, createProtobufSafeUpdater } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { listProjects } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ListProjectsResponseSchema } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Project } from '@ui/gen/api/v1alpha1/generated_pb';

export default {
  add: (projects: Project[]) => {
    queryClient.setQueriesData(
      {
        queryKey: createConnectQueryKey({ schema: listProjects, cardinality: 'finite' })
          // IMPORTANT: createConnectQueryKey returns falsy elements for filters so lets use only static identifiers
          .slice(0, 2),
        exact: false
      },
      createProtobufSafeUpdater(listProjects, (prev) => {
        let newProjects = [...(prev?.projects || [])];

        if (projects?.length > 0) {
          newProjects = newProjects.concat(projects);
        }

        return create(ListProjectsResponseSchema, {
          total: newProjects?.length,
          projects: newProjects
        });
      })
    );
  }
};
