import { describe, expect, test } from 'vitest';

import { PromotionRequest, Target } from '../../../../gen/api/v2/models';
import { targetRows } from '../../../project/targets/model/target-rows';
import {
  fleetStage,
  forStage as stageStatus,
  round,
  target
} from '../../../project/targets/model/test-fixtures';

import {
  assessedHealth,
  childPromotionName,
  governing,
  healthTally,
  neverPromoted,
  owningTargetSet,
  roundTally
} from './summary';

const stage = fleetStage('prod', [{ matchLabels: { env: 'prod' } }], 'current');

// rows builds the tab's rows the way the tab does: against this one Stage.
const rows = (targets: Target[], request?: PromotionRequest) =>
  targetRows(targets, [stage], { prod: request });

const governed = (
  name: string,
  status?: ReturnType<typeof stageStatus>,
  labels: Record<string, string> = { env: 'prod' }
) => target(name, labels, status ? { status: { stages: { prod: status } } } : {});

describe('governing()', () => {
  test("is what the tab's one Stage observes of a Target", () => {
    const [row] = rows([governed('t', stageStatus('current', 'Healthy', 'freight-new'))]);
    expect(governing(row)?.stage.metadata?.name).toBe('prod');
    expect(governing(row)?.health?.status).toBe('Healthy');
  });

  test('is nothing for a Target the Stage does not govern', () => {
    const [row] = rows([governed('t', undefined, { env: 'staging' })]);
    expect(governing(row)).toBeUndefined();
  });
});

describe('roundTally()', () => {
  test('counts the latest round by what it did to each Target', () => {
    const request = round('prod', ['a', 'b', 'c', 'd', 'e'], {
      targets: [
        { name: 'a', phase: 'Succeeded' },
        { name: 'b', phase: 'Failed' },
        { name: 'c', phase: 'Errored' },
        { name: 'd', phase: 'Running' },
        { name: 'e', phase: 'Aborted' }
      ]
    });
    expect(
      roundTally(
        rows(
          ['a', 'b', 'c', 'd', 'e'].map((name) => governed(name)),
          request
        )
      )
    ).toEqual({ inRound: 5, succeeded: 1, failed: 2, inFlight: 1, aborted: 1, notInRound: 0 });
  });

  test('a Target the round never named joined the Stage after it began', () => {
    const request = round('prod', ['a'], { targets: [{ name: 'a', phase: 'Succeeded' }] });
    expect(roundTally(rows([governed('a'), governed('late')], request))).toMatchObject({
      inRound: 1,
      succeeded: 1,
      notInRound: 1
    });
  });

  test('no round at all leaves everything out of it', () => {
    expect(roundTally(rows([governed('a'), governed('b')]))).toMatchObject({
      inRound: 0,
      notInRound: 2
    });
  });
});

describe('healthTally()', () => {
  test('counts what has been assessed apart from what has not', () => {
    const tally = healthTally(
      rows([
        governed('a', stageStatus('current', 'Healthy')),
        governed('b', stageStatus('current', 'Unhealthy')),
        governed('c', stageStatus('current')),
        governed('d')
      ])
    );
    // The breakdown counts the assessed Targets and no others, so it sums to
    // the figure the tile prints above it.
    expect(tally.assessed).toBe(2);
    expect(tally.byStatus).toEqual([
      { status: 'Healthy', count: 1 },
      { status: 'Unhealthy', count: 1 }
    ]);
    expect(tally.unassessed).toBe(2);
  });

  test('an unassessed fleet has nothing to break down', () => {
    const tally = healthTally(rows([governed('a'), governed('b')]));
    expect(tally).toEqual({ assessed: 0, byStatus: [], unassessed: 2 });
  });
});

describe('assessedHealth()', () => {
  test('is false until something has assessed a Target', () => {
    expect(assessedHealth(rows([governed('a'), governed('b', stageStatus('current'))]))).toBe(
      false
    );
  });

  test('is true as soon as one Target carries a status', () => {
    expect(
      assessedHealth(rows([governed('a'), governed('b', stageStatus('current', 'Unknown'))]))
    ).toBe(true);
  });
});

describe('neverPromoted()', () => {
  test('counts the Targets this Stage has never landed Freight on', () => {
    expect(
      neverPromoted(
        rows([
          governed('a', stageStatus('current', undefined, 'freight-new')),
          governed('b', stageStatus('old', undefined, 'freight-old')),
          governed('c')
        ])
      )
    ).toBe(1);
  });
});

describe('childPromotionName()', () => {
  const request = round('prod', ['a', 'b'], {
    targets: [
      { name: 'a', phase: 'Failed', promotion: 'prod.01.a' },
      { name: 'b', phase: 'Pending' }
    ]
  });

  test('names the Promotion the round made for a Target', () => {
    expect(childPromotionName(request, 'a')).toBe('prod.01.a');
  });

  test('is nothing before the round has made one, or for a Target it never named', () => {
    expect(childPromotionName(request, 'b')).toBeUndefined();
    expect(childPromotionName(request, 'elsewhere')).toBeUndefined();
    expect(childPromotionName(undefined, 'a')).toBeUndefined();
  });
});

describe('owningTargetSet()', () => {
  test('names the TargetSet a Target was generated from', () => {
    expect(
      owningTargetSet({
        metadata: {
          name: 't',
          ownerReferences: [
            { apiVersion: 'v1', kind: 'ConfigMap', name: 'not-it', uid: '1' },
            {
              apiVersion: 'ee.kargo.akuity.io/v1alpha1',
              kind: 'TargetSet',
              name: 'clusters',
              uid: '2'
            }
          ]
        }
      })
    ).toBe('clusters');
  });

  test('is nothing for a Target written by hand', () => {
    expect(owningTargetSet(target('t'))).toBeUndefined();
  });
});
