import { describe, expect, test } from 'vitest';

import { TargetStatusEntry } from '../../../../gen/api/v2/models';
import { targetRows } from '../../../project/targets/model/target-rows';
import { entry, fleetStage, health, target } from '../../../project/targets/model/test-fixtures';

import {
  MarkColors,
  attentionTally,
  byMark,
  freightMarking,
  healthMarking,
  statusMarking,
  tally
} from './marks';

// Colours are named rather than real, so a case says which vocabulary painted
// a square without turning into a hex comparison.
const colors: MarkColors = {
  state: {
    UpToDate: 'state-up-to-date',
    Behind: 'state-behind',
    Promoting: 'state-promoting',
    Pending: 'state-pending',
    Failed: 'state-failed',
    NotAttempted: 'state-not-attempted',
    NeverPromoted: 'state-never-promoted'
  },
  health: {
    Healthy: 'health-healthy',
    Progressing: 'health-progressing',
    Unhealthy: 'health-unhealthy',
    Unknown: 'health-unknown'
  },
  freight: ['freight-1', 'freight-2', 'freight-3']
};

const stage = fleetStage('prod', [{ matchLabels: { env: 'prod' } }], 'freight-new');

// rows builds the tab's rows the way the tab does: from this one Stage's
// statuses.
const rows = (entries: TargetStatusEntry[], healths: Record<string, string> = {}) =>
  targetRows(
    entries.map(({ target: name = '' }) =>
      target(
        name,
        { env: 'prod' },
        healths[name] ? { status: { stages: { prod: health(healths[name]) } } } : {}
      )
    ),
    entries,
    [stage]
  );

// freightLabel stands in for the alias a piece of Freight is named by.
const freightLabel = (name: string) => name.replace('freight-', '');

describe('statusMarking()', () => {
  const marks = statusMarking(colors);

  test('paints a Target by where it stands with the Stage, in the shared words', () => {
    const [failed, left, never, waiting] = rows([
      entry('a', 'prod', 'Failed'),
      entry('b', 'prod', 'NotAttempted'),
      entry('c', 'prod', 'NeverPromoted'),
      entry('d', 'prod', 'Pending')
    ]);
    expect(marks.mark(failed)).toMatchObject({
      key: 'Failed',
      label: 'Failed',
      color: 'state-failed',
      attention: true
    });
    expect(marks.mark(left)).toMatchObject({
      key: 'NotAttempted',
      label: 'Not attempted',
      color: 'state-not-attempted'
    });
    expect(marks.mark(never)).toMatchObject({ key: 'NeverPromoted', hollow: true });
    expect(marks.mark(waiting)).toMatchObject({ key: 'Pending', label: 'Pending' });
  });

  test('a Target a retry left out keeps the state the rollout before left it in', () => {
    // Not "never promoted", and not "not in round": the server reads the
    // whole history, not the latest round.
    const [row] = rows([entry('a', 'prod', 'NotAttempted', { rollout: 3, freightName: 'f-old' })]);
    expect(marks.mark(row).key).toBe('NotAttempted');
  });

  test('the legend leads with what is settled and ends with what nothing reached', () => {
    expect(marks.legend.map(({ key }) => key)).toEqual([
      'UpToDate',
      'Behind',
      'Promoting',
      'Pending',
      'Failed',
      'NotAttempted',
      'NeverPromoted'
    ]);
  });
});

describe('healthMarking()', () => {
  const marks = healthMarking(colors);

  test('paints a Target by its health for this Stage', () => {
    const [healthy, sick] = rows([entry('a', 'prod', 'UpToDate'), entry('b', 'prod', 'UpToDate')], {
      a: 'Healthy',
      b: 'Unhealthy'
    });
    expect(marks.mark(healthy)).toMatchObject({ key: 'Healthy', color: 'health-healthy' });
    expect(marks.mark(sick)).toMatchObject({ key: 'Unhealthy', attention: true });
  });

  test('a Target nothing has assessed is Unknown', () => {
    const [row] = rows([entry('a', 'prod', 'UpToDate')]);
    expect(marks.mark(row)).toMatchObject({ key: 'Unknown', color: 'health-unknown' });
  });
});

describe('freightMarking()', () => {
  const fleet = () =>
    rows([
      entry('a', 'prod', 'UpToDate', { freightName: 'freight-new' }),
      entry('b', 'prod', 'UpToDate', { freightName: 'freight-new' }),
      entry('c', 'prod', 'Behind', { freightName: 'freight-old' }),
      entry('d', 'prod', 'NeverPromoted')
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
      { key: 'NoFreight', label: 'none', color: 'state-never-promoted', attention: undefined }
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
  const fleet = () =>
    rows([
      entry('a', 'prod', 'UpToDate'),
      entry('b', 'prod', 'UpToDate'),
      entry('c', 'prod', 'Failed')
    ]);

  test('counts in legend order, leaving out what nothing is in', () => {
    expect(
      tally(fleet(), statusMarking(colors)).map(({ mark, count }) => [mark.key, count])
    ).toEqual([
      ['UpToDate', 2],
      ['Failed', 1]
    ]);
  });

  test('attentionTally is the largest category worth looking at', () => {
    expect(attentionTally(fleet(), statusMarking(colors))).toMatchObject({ count: 1 });
  });

  test('a fleet with nothing wrong has nothing to draw attention to', () => {
    expect(
      attentionTally(rows([entry('a', 'prod', 'UpToDate')]), statusMarking(colors))
    ).toBeUndefined();
  });
});

describe('byMark()', () => {
  test('orders trouble first, then by the legend, then by name', () => {
    const fleet = rows([
      entry('won', 'prod', 'UpToDate'),
      entry('also-won', 'prod', 'UpToDate'),
      entry('lost', 'prod', 'Failed'),
      entry('waiting', 'prod', 'Pending')
    ]);
    expect(
      [...fleet].sort(byMark(statusMarking(colors))).map((row) => row.target.metadata?.name)
    ).toEqual(['lost', 'also-won', 'won', 'waiting']);
  });
});
