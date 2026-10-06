import { describe, expect, test } from 'vitest';

import { Stage, Target, V1LabelSelector } from '@ui/gen/api/v2/models';

import {
  UNLABELED,
  groupTargetRows,
  labelKeys,
  matchesLabelSelector,
  matchesSearch,
  stageGovernsTarget,
  targetAwareStages,
  targetRows
} from './targets-utils';

const target = (
  name: string,
  labels?: Record<string, string>,
  partial: Partial<Target> = {}
): Target => ({
  metadata: { name, labels },
  ...partial
});

const stage = (name: string, partial: Partial<Stage> = {}): Stage => ({
  metadata: { name },
  ...partial
});

const fleetStage = (name: string, selectors: V1LabelSelector[]) =>
  stage(name, { spec: { requestedFreight: [], targets: { selectors } } } as Partial<Stage>);

describe('matchesLabelSelector()', () => {
  test('an empty selector matches everything', () => {
    expect(matchesLabelSelector({}, {})).toBe(true);
    expect(matchesLabelSelector(undefined, { a: 'b' })).toBe(true);
  });

  test('matchLabels are ANDed', () => {
    const selector = { matchLabels: { region: 'us', tier: 'prod' } };
    expect(matchesLabelSelector(selector, { region: 'us', tier: 'prod' })).toBe(true);
    expect(matchesLabelSelector(selector, { region: 'us' })).toBe(false);
  });

  test.each([
    ['In', ['us', 'eu'], { region: 'us' }, true],
    ['In', ['us', 'eu'], { region: 'ap' }, false],
    ['In', ['us'], {}, false],
    ['NotIn', ['us'], { region: 'eu' }, true],
    ['NotIn', ['us'], {}, true],
    ['NotIn', ['us'], { region: 'us' }, false],
    ['Exists', undefined, { region: 'x' }, true],
    ['Exists', undefined, {}, false],
    ['DoesNotExist', undefined, {}, true],
    ['DoesNotExist', undefined, { region: 'x' }, false],
    ['Bogus', undefined, { region: 'x' }, false]
  ])('%s %j against %j -> %s', (operator, values, labels, expected) => {
    expect(
      matchesLabelSelector(
        { matchExpressions: [{ key: 'region', operator, values }] },
        labels as Record<string, string>
      )
    ).toBe(expected);
  });
});

describe('stageGovernsTarget()', () => {
  test('a classic Stage governs nothing', () => {
    expect(stageGovernsTarget(stage('classic'), target('t', { region: 'us' }))).toBe(false);
  });

  test('an empty selector list governs nothing', () => {
    expect(stageGovernsTarget(fleetStage('fleet', []), target('t', { region: 'us' }))).toBe(false);
  });

  test('selectors are a union', () => {
    const fleet = fleetStage('fleet', [
      { matchLabels: { region: 'us' } },
      { matchLabels: { region: 'eu' } }
    ]);
    expect(stageGovernsTarget(fleet, target('a', { region: 'us' }))).toBe(true);
    expect(stageGovernsTarget(fleet, target('b', { region: 'eu' }))).toBe(true);
    expect(stageGovernsTarget(fleet, target('c', { region: 'ap' }))).toBe(false);
  });
});

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

describe('labelKeys()', () => {
  test('collects distinct keys, sorted, minus Kargo-internal ones', () => {
    expect(
      labelKeys([
        target('a', { region: 'us', 'kargo.akuity.io/shard': 'x' }),
        target('b', { env: 'prod', region: 'eu' }),
        target('c')
      ])
    ).toEqual(['env', 'region']);
  });
});

describe('targetRows()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }]);
  const all = fleetStage('all-stage', [{}]);
  const classic = stage('classic');

  test('one row per Target with its governing Stages, both sorted by name', () => {
    const rows = targetRows(
      [target('us-west-2', { region: 'us' }), target('eu-central-1', { region: 'eu' })],
      [us, classic, all]
    );
    expect(rows.map((r) => r.target.metadata?.name)).toEqual(['eu-central-1', 'us-west-2']);
    expect(rows[0].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage']);
    expect(rows[1].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage', 'us-stage']);
  });

  test('an ungoverned Target still gets a row', () => {
    const [row] = targetRows([target('orphan')], [us]);
    expect(row.stages).toEqual([]);
  });

  test('carries the Target health recorded for each Stage', () => {
    const t = target(
      't',
      { region: 'us' },
      {
        status: { stages: { 'us-stage': { health: { status: 'Unhealthy' } } } }
      }
    );
    const [row] = targetRows([t], [us, all]);
    const byStage = Object.fromEntries(row.stages.map((s) => [s.stage.metadata?.name, s.health]));
    expect(byStage['us-stage']?.status).toBe('Unhealthy');
    expect(byStage['all-stage']).toBeUndefined();
  });
});

describe('groupTargetRows()', () => {
  const rows = targetRows(
    [
      target('us-west-2', { region: 'us' }),
      target('eu-central-1', { region: 'eu' }),
      target('orphan'),
      target('us-east-1', { region: 'us' })
    ],
    []
  );

  test('without a key, one group', () => {
    expect(groupTargetRows(rows)).toHaveLength(1);
    expect(groupTargetRows(rows)[0].rows).toHaveLength(4);
  });

  test('by label, unlabeled last', () => {
    const groups = groupTargetRows(rows, 'region');
    expect(groups.map((g) => g.value)).toEqual(['eu', 'us', UNLABELED]);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['us-east-1', 'us-west-2']);
  });

  test('no rows yields no groups', () => {
    expect(groupTargetRows([])).toEqual([]);
    expect(groupTargetRows([], 'region')).toEqual([]);
  });
});

describe('matchesSearch()', () => {
  const us = fleetStage('prod-us', [{ matchLabels: { region: 'us' } }]);
  const [row] = targetRows([target('us-east-1', { region: 'us', tier: 'canary' })], [us]);

  test.each([
    ['', true],
    ['east', true],
    ['REGION=us', true],
    ['canary', true],
    ['prod-us', true],
    ['eu', false]
  ])('%j -> %s', (needle, expected) => {
    expect(matchesSearch(row, needle)).toBe(expected);
  });
});
