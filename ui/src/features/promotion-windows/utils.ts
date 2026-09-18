import { stringify } from 'yaml';

import { ClusterConfig, ProjectConfig } from '@ui/gen/api/v2/models';

export const projectConfigGen = {
  v1alpha1: (def: ProjectConfig) =>
    stringify({
      apiVersion: 'kargo.akuity.io/v1alpha1',
      kind: 'ProjectConfig',
      ...def
    })
};

export const clusterConfigGen = {
  v1alpha1: (def: ClusterConfig) =>
    stringify({
      apiVersion: 'kargo.akuity.io/v1alpha1',
      kind: 'ClusterConfig',
      ...def
    })
};
