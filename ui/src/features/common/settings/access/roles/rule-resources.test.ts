import { describe, expect, it } from 'vitest';

import { rulesForResources } from './rule-resources';

describe('role resource groups', () => {
  it('qualifies Events as core even when events.k8s.io is also installed', () => {
    expect(rulesForResources({ resources: ['events'], verbs: ['get', 'list'] })).toEqual([
      { apiGroups: [''], resources: ['events'], verbs: ['get', 'list'] }
    ]);
  });

  it('keeps mixed resource permissions in their own groups and preserves names/verbs', () => {
    const rules = rulesForResources({
      resources: ['events', 'stages', 'analysisruns', 'roles'],
      verbs: ['get'],
      resourceNames: ['example']
    });
    expect(rules.map((rule) => rule.apiGroups)).toEqual([
      [''],
      ['kargo.akuity.io'],
      ['argoproj.io'],
      ['rbac.authorization.k8s.io']
    ]);
    for (const rule of rules) {
      expect(rule.verbs).toEqual(['get']);
      expect(rule.resourceNames).toEqual(['example']);
    }
  });

  it('leaves custom resources to API discovery instead of guessing a group', () => {
    expect(rulesForResources({ resources: ['widgets'], verbs: ['get'] })).toEqual([
      { resources: ['widgets'], verbs: ['get'] }
    ]);
  });
});
