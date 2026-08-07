import { queryClient } from '@ui/config/query-client';
import { RolloutsAnalysisTemplate } from '@ui/gen/api/v2/models';
import {
  getListAnalysisTemplatesQueryKey,
  listAnalysisTemplatesResponse
} from '@ui/gen/api/v2/verifications/verifications';

export default {
  add: (project: string, templates: RolloutsAnalysisTemplate[]) => {
    queryClient.setQueriesData<listAnalysisTemplatesResponse>(
      {
        queryKey: getListAnalysisTemplatesQueryKey(project),
        exact: false
      },
      (prev) => {
        if (!prev || !templates?.length) {
          return prev;
        }

        return {
          ...prev,
          data: {
            ...prev.data,
            items: [...(prev.data.items || []), ...templates]
          }
        };
      }
    );
  }
};
