import { describe, expect, test } from 'vitest';

import { PromotionRequest, TargetSummary } from '../../../../gen/api/v2/models';
import { targetRows } from '../../../project/targets/model/target-rows';
import {
  fleetStage,
  forStage as stageStatus,
  round,
  target
} from '../../../project/targets/model/test-fixtures';

import {
  MarkColors,
  attentionTally,
  byMark,
  freightMarking,
  healthMarking,
  promotionMarking,
  tally
} from './marks';

// Colours are named rather than real, so a case says which vocabulary painted
// a square without turning into a hex comparison.
const colors: MarkColors = {
  phase: {
    Succeeded: 'phase-succeeded',
    Running: 'phase-running',
    Pending: 'phase-pending',
    Failed: 'phase-failed',
    Errored: 'phase-errored',
    Aborted: 'phase-aborted'
  },
  state: {
    Healthy: 'state-healthy',
    Behind: 'state-behind',
    Progressing: 'state-progressing',
    Unhealthy: 'state-unhealthy',
    Unknown: 'state-unknown',
    NotPromoted: 'state-not-promoted'
  },
  freight: ['freight-1', 'freight-2', 'freight-3']
};

const stage = fleetStage('prod', [{ matchLabels: { env: 'prod' } }], 'current');

const rows = (targets: TargetSummary[], request?: PromotionRequest) =>
  targetRows(targets, [stage], { prod: request });

const governed = (name: string, status?: ReturnType<typeof stageStatus>) =>
  target(name, { env: 'prod' }, { prod: status ?? {} });

// freightLabel stands in for the alias a piece of Freight is named by.
const freightLabel = (name: string) => name.replace('freight-', '');

describe('promotionMarking()', () => {
  const marks = promotionMarking(colors);

  test('paints a Target by what the round did to it', () => {
    const request = round('prod', ['won', 'lost'], {
      targets: [
        { name: 'won', phase: 'Succeeded' },
        { name: 'lost', phase: 'Failed' }
      ]
    });
    const [lost, won] = rows([governed('won'), governed('lost')], request);
    expect(marks.mark(won)).toMatchObject({ key: 'Succeeded', color: 'phase-succeeded' });
    expect(marks.mark(lost)).toMatchObject({
      key: 'Failed',
      color: 'phase-failed',
      attention: true
    });
  });

  test('a Target the round never named is read by whether it has Freight', () => {
    const request = round('prod', ['named'], { targets: [{ name: 'named', phase: 'Succeeded' }] });
    const [late, never] = rows(
      [governed('late', stageStatus('old', undefined, 'freight-old')), governed('never')],
      request
    );
    expect(marks.mark(late).key).toBe('NotInRound');
    expect(marks.mark(never)).toMatchObject({ key: 'NeverPromoted', hollow: true });
  });

  test('a blocked round did nothing to anything, and says so once', () => {
    // roundBlock fires on a round that cannot fan out at all, so targetRows
    // leaves every Target phaseless -- which must not read as "never named".
    const blockedMarks = promotionMarking(colors, true);
    const [row] = rows([governed('a', stageStatus('current', undefined, 'freight-new'))]);
    expect(blockedMarks.legend.map(({ key }) => key)).toEqual(['NotAttempted']);
    expect(blockedMarks.mark(row)).toMatchObject({ key: 'NotAttempted', label: 'Not attempted' });
  });

  test('the legend leads with what went well and ends with what was never tried', () => {
    expect(marks.legend.map(({ key }) => key)).toEqual([
      'Succeeded',
      'Running',
      'Pending',
      'Failed',
      'Errored',
      'Aborted',
      'NotInRound',
      'NeverPromoted'
    ]);
  });
});

describe('healthMarking()', () => {
  const marks = healthMarking(colors);

  test('paints a Target by its health for this Stage', () => {
    const [healthy, sick] = rows([
      governed('a', stageStatus('current', 'Healthy')),
      governed('b', stageStatus('current', 'Unhealthy'))
    ]);
    expect(marks.mark(healthy)).toMatchObject({ key: 'Healthy', color: 'state-healthy' });
    expect(marks.mark(sick)).toMatchObject({ key: 'Unhealthy', attention: true });
  });

  test('a Target nothing has assessed is Unknown', () => {
    const [row] = rows([governed('a')]);
    expect(marks.mark(row)).toMatchObject({ key: 'Unknown', color: 'state-unknown' });
  });
});

describe('freightMarking()', () => {
  const fleet = () =>
    rows([
      governed('a', stageStatus('current', undefined, 'freight-new')),
      governed('b', stageStatus('current', undefined, 'freight-new')),
      governed('c', stageStatus('old', undefined, 'freight-old')),
      governed('d')
    ]);

  test("the Stage's own Freight leads, and everything else is a Target left behind", () => {
    const marks = freightMarking(fleet(), ['freight-new'], colors, freightLabel);
    expect(
      marks.legend.map(({ key, label, color, attention }) => ({
        key,
        label,
        color,
        attention
      }))
    ).toEqual([
      { key: 'freight-new', label: 'new', color: 'freight-1', attention: false },
      { key: 'freight-old', label: 'old', color: 'freight-2', attention: true },
      { key: 'NoFreight', label: 'none', color: 'state-not-promoted', attention: undefined }
    ]);
  });

  test('a Target with no Freight falls into the hollow category', () => {
    const marks = freightMarking(fleet(), ['freight-new'], colors, freightLabel);
    const none = fleet().find((row) => !row.freight.length);
    expect(marks.mark(none!)).toMatchObject({ key: 'NoFreight', hollow: true });
  });

  test('with the Stage on nothing yet, the commonest Freight leads', () => {
    const marks = freightMarking(fleet(), [], colors, freightLabel);
    expect(marks.legend.map(({ key }) => key)).toEqual(['freight-new', 'freight-old', 'NoFreight']);
    // Nothing is current, so every Freight on the fleet is something to look at.
    expect(marks.legend[0].attention).toBe(true);
  });
});

describe('tally()', () => {
  const request = round('prod', ['a', 'b', 'c'], {
    targets: [
      { name: 'a', phase: 'Succeeded' },
      { name: 'b', phase: 'Succeeded' },
      { name: 'c', phase: 'Failed' }
    ]
  });
  const fleet = () =>
    rows(
      ['a', 'b', 'c'].map((name) => governed(name)),
      request
    );

  test('counts in legend order, leaving out what nothing is in', () => {
    expect(
      tally(fleet(), promotionMarking(colors)).map(({ mark, count }) => [mark.key, count])
    ).toEqual([
      ['Succeeded', 2],
      ['Failed', 1]
    ]);
  });

  test('attentionTally is the largest category worth looking at', () => {
    expect(attentionTally(fleet(), promotionMarking(colors))).toMatchObject({ count: 1 });
  });

  test('a fleet with nothing wrong has nothing to draw attention to', () => {
    const clean = round('prod', ['a'], { targets: [{ name: 'a', phase: 'Succeeded' }] });
    expect(attentionTally(rows([governed('a')], clean), promotionMarking(colors))).toBeUndefined();
  });
});

describe('byMark()', () => {
  test('orders trouble first, then by the legend, then by name', () => {
    const request = round('prod', ['won', 'also-won', 'lost', 'waiting'], {
      targets: [
        { name: 'won', phase: 'Succeeded' },
        { name: 'also-won', phase: 'Succeeded' },
        { name: 'lost', phase: 'Failed' },
        { name: 'waiting', phase: 'Pending' }
      ]
    });
    const fleet = rows(
      ['won', 'also-won', 'lost', 'waiting'].map((name) => governed(name)),
      request
    );
    expect([...fleet].sort(byMark(promotionMarking(colors))).map((row) => row.target.name)).toEqual(
      ['lost', 'also-won', 'won', 'waiting']
    );
  });
});
