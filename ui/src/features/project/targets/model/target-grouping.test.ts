import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { Facets, matchesFacets, stateFacet } from './target-facets';
import {
  STAGE_GROUPING,
  UNGOVERNED,
  UNLABELED,
  groupAxis,
  groupHeading,
  filterGroupedRows,
  groupTargetRows,
  groupTitle,
  groupedRows,
  splitKeys
} from './target-grouping';
import { stateCounts, targetRows } from './target-rows';
import { fleetStage, forStage, stage, target } from './test-fixtures';

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

  test('an empty label value groups with the unlabeled, not into a blank group', () => {
    const withEmpty = targetRows(
      [target('a', { region: '' }), target('b', { region: 'us' }), target('c')],
      []
    );
    const groups = groupTargetRows(withEmpty, 'region');
    expect(groups.map((g) => g.value)).toEqual(['us', UNLABELED]);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['a', 'c']);
  });

  test('no rows yields no groups', () => {
    expect(groupTargetRows([])).toEqual([]);
    expect(groupTargetRows([], 'region')).toEqual([]);
  });
});

describe('groupTargetRows() by Stage', () => {
  // dev feeds prod, so dev comes first even though prod sorts ahead by name.
  const dev = fleetStage('dev', [{ matchLabels: { tier: 'all' } }], 'new');
  const prod: Stage = {
    ...fleetStage('prod', [{ matchLabels: { tier: 'all' } }], 'new'),
    spec: {
      requestedFreight: [
        { origin: { kind: 'Warehouse', name: 'w' }, sources: { stages: ['dev'] } }
      ],
      targets: { selectors: [{ matchLabels: { tier: 'all' } }] }
    }
  };
  const rows = targetRows(
    [
      target(
        'shared',
        { tier: 'all' },
        {
          status: {
            stages: {
              dev: forStage('new', 'Healthy', 'f-new'),
              prod: forStage('old', 'Unhealthy', 'f-old')
            }
          }
        }
      ),
      target('orphan')
    ],
    [prod, dev]
  );
  const groups = groupTargetRows(rows, STAGE_GROUPING, [prod, dev]);

  test('a Target governed by several Stages lands in each, in pipeline order', () => {
    expect(groups.map((g) => g.value)).toEqual(['dev', 'prod', UNGOVERNED]);
    expect(groups[0].rows.map((r) => r.target.metadata?.name)).toEqual(['shared']);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['shared']);
  });

  test('each group reads the Target as its own Stage sees it', () => {
    // Across both Stages the Target is Unhealthy; dev alone sees it Healthy.
    expect(rows[1].state).toBe('Unhealthy');
    expect(groups[0].rows[0].state).toBe('Healthy');
    expect(groups[0].rows[0].freight).toEqual(['f-new']);
    expect(groups[1].rows[0].state).toBe('Unhealthy');
    expect(groups[1].rows[0].freight).toEqual(['f-old']);
  });

  test('a scoped row still knows every Stage that governs its Target', () => {
    // The reading narrows to one Stage; the Stages column and the detail panel
    // must still show the Stage the trouble is on.
    expect(groups[0].rows[0].scopedTo?.stage.metadata?.name).toBe('dev');
    expect(groups[0].rows[0].stages.map((s) => s.stage.metadata?.name)).toEqual(['dev', 'prod']);
  });

  test('the ungoverned come last, unscoped', () => {
    expect(groups[2].rows.map((r) => r.target.metadata?.name)).toEqual(['orphan']);
    expect(groups[2].rows[0].stages).toEqual([]);
    expect(groups[2].rows[0].scopedTo).toBeUndefined();
  });

  test('a Stage no row is governed by gets no group', () => {
    const idle = fleetStage('idle', [{ matchLabels: { tier: 'none' } }]);
    const withIdle = targetRows([target('a', { tier: 'all' })], [dev, idle]);
    expect(groupTargetRows(withIdle, STAGE_GROUPING, [dev, idle]).map((g) => g.value)).toEqual([
      'dev'
    ]);
  });

  test('no rows yields no groups', () => {
    expect(groupTargetRows([], STAGE_GROUPING, [dev, stage('classic')])).toEqual([]);
  });

  // The regression the ordering exists for: faceting runs inside the groups,
  // so the filter and the rows it leaves behind cannot disagree.
  describe('faceting inside the groups', () => {
    const facetedGroups = (states: Facets['states']) =>
      filterGroupedRows(groups, (row) => matchesFacets(row, { labels: {}, states }));

    test('ticking Healthy keeps the Stage that sees the Target healthy', () => {
      const healthy = facetedGroups(['Healthy']);
      expect(healthy.map((g) => g.value)).toEqual(['dev']);
      expect(healthy[0].rows.map((r) => r.target.metadata?.name)).toEqual(['shared']);
    });

    test('ticking Unhealthy keeps only the Stage that sees it unhealthy', () => {
      const unhealthy = facetedGroups(['Unhealthy']);
      expect(unhealthy.map((g) => g.value)).toEqual(['prod']);
      expect(unhealthy[0].rows[0].state).toBe('Unhealthy');
    });

    test('the facet counts agree with what each group shows', () => {
      expect(stateFacet(groupedRows(groups), { labels: {}, states: [] }).Healthy).toBe(
        stateCounts(groups[0].rows).Healthy
      );
    });

    test('the summary counts the readings the groups hold between them', () => {
      const counts = stateCounts(groupedRows(groups));
      expect(counts.Healthy).toBe(1);
      expect(counts.Unhealthy).toBe(1);
      expect(groupedRows(groups)).toHaveLength(3);
    });
  });
});

describe('groupAxis()', () => {
  test('names the Stage axis in words and a label key as itself', () => {
    expect(groupAxis(STAGE_GROUPING)).toBe('Stage');
    expect(groupAxis('geo')).toBe('geo');
  });
});

describe('groupHeading() and groupTitle()', () => {
  const group = (value: string) => ({ value, rows: [] });

  test.each([
    [group('us'), 'geo', 'us', 'geo=us'],
    [group(UNLABELED), 'geo', 'no geo label', 'no geo label'],
    [group(''), undefined, 'All Targets', 'All Targets'],
    [group('prod'), STAGE_GROUPING, 'prod', 'Stage prod'],
    [group(UNGOVERNED), STAGE_GROUPING, 'no Stage', 'no Stage'],
    // Each sentinel is only read on the axis that can produce it: a Target may
    // carry the sentinel's own text as a label value, and that is a label group
    // like any other.
    [group(UNGOVERNED), 'geo', UNGOVERNED, `geo=${UNGOVERNED}`],
    [group(UNLABELED), STAGE_GROUPING, UNLABELED, `Stage ${UNLABELED}`]
  ])('%j by %j', (target, key, heading, title) => {
    expect(groupHeading(target, key)).toBe(heading);
    expect(groupTitle(target, key)).toBe(title);
  });
});

describe('splitKeys()', () => {
  test('a short list all fits on the row', () => {
    expect(splitKeys(['geo', 'cloud', 'tier'])).toEqual({
      inline: ['geo', 'cloud', 'tier'],
      overflow: []
    });
  });

  test('the tail past four folds into the dropdown', () => {
    expect(splitKeys(['geo', 'cloud', 'tier', 'env', 'stage', 'shard'])).toEqual({
      inline: ['geo', 'cloud', 'tier', 'env'],
      overflow: ['stage', 'shard']
    });
  });

  test('a long key folds in however early it comes', () => {
    expect(splitKeys(['app.kubernetes.io/part-of', 'geo', 'cloud'])).toEqual({
      inline: ['geo', 'cloud'],
      overflow: ['app.kubernetes.io/part-of']
    });
  });

  test('short keys further down are promoted past it', () => {
    expect(
      splitKeys(['app.kubernetes.io/part-of', 'geo', 'cloud', 'tier', 'env', 'stage'])
    ).toEqual({
      inline: ['geo', 'cloud', 'tier', 'env'],
      overflow: ['app.kubernetes.io/part-of', 'stage']
    });
  });

  test('when every key is long the order stands', () => {
    const keys = ['app.kubernetes.io/part-of', 'app.kubernetes.io/component'];
    expect(splitKeys(keys)).toEqual({ inline: keys, overflow: [] });
  });

  test('no keys, nothing to show', () => {
    expect(splitKeys([])).toEqual({ inline: [], overflow: [] });
  });
});
