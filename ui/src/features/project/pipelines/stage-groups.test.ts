import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { getStageGroup, groupStages, STAGE_GROUP_LABEL, summarizeGroup } from './stage-groups';

const stage = (name: string, group?: string, health?: string): Stage =>
  ({
    metadata: { name, labels: group ? { [STAGE_GROUP_LABEL]: group } : {} },
    status: health ? { health: { status: health } } : {}
  }) as Stage;

const stages = [
  stage('dev-a-1', 'dev'),
  stage('dev-a-2', 'dev'),
  stage('dev-systest-1', 'systest'),
  stage('staging'),
  stage('prod')
];

describe('groupStages', () => {
  test('groups by label value and collapses every group by default', () => {
    const { members, collapsed } = groupStages(stages);
    expect(Object.keys(members).sort()).toEqual(['dev', 'systest']);
    expect(members.dev.map((s) => s.metadata?.name)).toEqual(['dev-a-1', 'dev-a-2']);
    expect([...collapsed].sort()).toEqual(['dev', 'systest']);
  });

  test('stages without the label are never grouped', () => {
    const { members } = groupStages([stage('staging'), stage('prod')]);
    expect(members).toEqual({});
    expect(getStageGroup(stage('prod'))).toBeUndefined();
  });

  test('an expanded group is not collapsed', () => {
    const { collapsed } = groupStages(stages, ['dev']);
    expect([...collapsed]).toEqual(['systest']);
  });

  test('a search hit inside a group expands it', () => {
    const { collapsed } = groupStages(stages, [], ' DEV-A-2 ');
    expect([...collapsed]).toEqual(['systest']);
  });

  test('a search with no hit in a group leaves it collapsed', () => {
    const { collapsed } = groupStages(stages, [], 'prod');
    expect([...collapsed].sort()).toEqual(['dev', 'systest']);
  });
});

describe('summarizeGroup', () => {
  test('counts healthy, unhealthy and other', () => {
    expect(
      summarizeGroup([
        stage('a', 'g', 'Healthy'),
        stage('b', 'g', 'Healthy'),
        stage('c', 'g', 'Unhealthy'),
        stage('d', 'g', 'Degraded'),
        stage('e', 'g', 'Progressing'),
        stage('f', 'g')
      ])
    ).toEqual({ total: 6, healthy: 2, unhealthy: 2, other: 2 });
  });
});
