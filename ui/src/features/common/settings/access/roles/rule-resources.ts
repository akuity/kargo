import { V1PolicyRule } from '@ui/gen/api/v2/models';

export const resourceGroups: Record<string, string> = {
  configmaps: '',
  events: '',
  secrets: '',
  serviceaccounts: '',
  rolebindings: 'rbac.authorization.k8s.io',
  roles: 'rbac.authorization.k8s.io',
  freights: 'kargo.akuity.io',
  stages: 'kargo.akuity.io',
  promotions: 'kargo.akuity.io',
  warehouses: 'kargo.akuity.io',
  analysisruns: 'argoproj.io',
  analysistemplates: 'argoproj.io'
};

export const rulesForResources = (rule: V1PolicyRule): V1PolicyRule[] =>
  (rule.resources || []).map((resource) => ({
    ...rule,
    resources: [resource],
    // Disambiguate suggested resources, especially core Events. Preserve
    // discovery-based resolution for custom resources typed into the form.
    ...(Object.hasOwn(resourceGroups, resource) ? { apiGroups: [resourceGroups[resource]] } : {})
  }));
