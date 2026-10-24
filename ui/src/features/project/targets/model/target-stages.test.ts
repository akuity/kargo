import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { pipelineOrder, targetAwareStages } from './target-stages';
import { fleetStage, stage } from './test-fixtures';

describe('targetAwareStages()', () => {
  test('keeps only target-aware Stages, sorted by name', () => {
    const result = targetAwareStages([
      fleetStage('zeta', []),
      stage('classic'),
      fleetStage('alpha', [])
    ]);
    expect(result.map((s) => s.metadata?.name)).toEqual(['alpha', 'zeta']);
  });
});

describe('pipelineOrder()', () => {
  // dev -> staging -> {prod-us, prod-eu}, as a pipeline draws it left to right.
  const from = (name: string, ...sources: string[]): Stage =>
    stage(name, {
      spec: {
        requestedFreight: [
          { origin: { kind: 'Warehouse', name: 'w' }, sources: { stages: sources } }
        ]
      }
    } as Partial<Stage>);

  const pipeline = [
    from('prod-us', 'staging'),
    from('dev'),
    from('prod-eu', 'staging'),
    from('staging', 'dev')
  ];

  test('upstream first, with one column in name order', () => {
    expect(pipelineOrder(pipeline).map((s) => s.metadata?.name)).toEqual([
      'dev',
      'staging',
      'prod-eu',
      'prod-us'
    ]);
  });

  test('ranks over every Stage, so a classic Stage upstream still counts', () => {
    // `staging` is not in the subset being ordered, but it still pushes the
    // prod Stages a column further along than a Stage fed straight from a
    // Warehouse.
    const subset = [from('prod-us', 'staging'), from('direct')];
    expect(pipelineOrder(subset, pipeline.concat(subset)).map((s) => s.metadata?.name)).toEqual([
      'direct',
      'prod-us'
    ]);
  });

  test('a cycle does not hang the page', () => {
    const cyclic = [from('a', 'b'), from('b', 'a')];
    expect(pipelineOrder(cyclic).map((s) => s.metadata?.name)).toHaveLength(2);
  });

  test('a Stage requesting nothing is a root', () => {
    expect(pipelineOrder([stage('lonely')]).map((s) => s.metadata?.name)).toEqual(['lonely']);
  });
});
