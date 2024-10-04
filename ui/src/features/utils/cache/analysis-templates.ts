import { createConnectQueryKey, createProtobufSafeUpdater } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { AnalysisTemplate } from '@ui/gen/rollouts/api/v1alpha1/generated_pb';
import { listAnalysisTemplates } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

export default {
  add: (project: string, templates: AnalysisTemplate[]) => {
    queryClient.setQueriesData(
      {
        queryKey: createConnectQueryKey(listAnalysisTemplates, { project }),
        exact: false
      },
      createProtobufSafeUpdater(listAnalysisTemplates, (prev) => {
        let newTemplates = [...(prev?.analysisTemplates || [])];

        if (templates?.length > 0) {
          newTemplates = newTemplates.concat(templates);
        }

        return {
          ...prev,
          analysisTemplates: newTemplates
        };
      })
    );
  }
};
