// cache invalidation source-of-truth

import { createConnectQueryKey, createProtobufSafeUpdater } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { AnalysisTemplate } from '@ui/gen/rollouts/api/v1alpha1/generated_pb';
import {
  listAnalysisTemplates,
  listProjects,
  listStages,
  listWarehouses
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Project, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

export const queryCache = {
  project: {
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
  },
  warehouse: {
    add: (project: string, warehouses: Warehouse[]) => {
      queryClient.setQueriesData(
        {
          queryKey: createConnectQueryKey(listWarehouses, { project }),
          exact: false
        },
        createProtobufSafeUpdater(listWarehouses, (prev) => {
          const newWarehouses = [...(prev?.warehouses || [])];

          if (warehouses?.length > 0) {
            for (const warehouse of warehouses) {
              if (!newWarehouses.find((w) => w?.metadata?.name === warehouse?.metadata?.name)) {
                newWarehouses.push(warehouse);
              }
            }
          }

          return {
            ...prev,
            warehouses: newWarehouses
          };
        })
      );
    }
  },
  stage: {
    add: (project: string, stages: Stage[]) => {
      queryClient.setQueriesData(
        {
          queryKey: createConnectQueryKey(listStages, { project }),
          exact: false
        },
        createProtobufSafeUpdater(listStages, (prev) => {
          const newStages = [...(prev?.stages || [])];

          if (stages?.length > 0) {
            for (const stage of stages) {
              if (!newStages.find((s) => s?.metadata?.name === stage?.metadata?.name)) {
                newStages.push(stage);
              }
            }
          }

          return {
            ...prev,
            stages: newStages
          };
        })
      );
    }
  },
  analysisTemplates: {
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
  },
  promotion: {}
};
