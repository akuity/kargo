import { describe, expect, it } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { resourceMembership } from './node-indexer';

const stage = (name: string, uid = name): Stage => ({ metadata: { name, uid }, spec: {} });

describe('graph resource membership', () => {
  it('changes when a resource is deleted, added, or replaced with a new UID', () => {
    const before = resourceMembership([stage('dev'), stage('prod')]);
    expect(resourceMembership([stage('dev')])).not.toBe(before);
    expect(resourceMembership([stage('dev'), stage('prod'), stage('qa')])).not.toBe(before);
    expect(resourceMembership([stage('dev'), stage('prod', 'replacement')])).not.toBe(before);
  });

  it('does not relayout for ordering or status-only changes', () => {
    const before = resourceMembership([stage('dev'), stage('prod')]);
    const updated: Stage = { ...stage('dev'), status: { observedGeneration: 2 } };
    expect(resourceMembership([stage('prod'), updated])).toBe(before);
  });
});
