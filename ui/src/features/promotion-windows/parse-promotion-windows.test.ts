import { describe, expect, it } from 'vitest';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { parsePromotionWindows } from './parse-promotion-windows';

const makeWindow = (overrides: Partial<PromotionWindow>): PromotionWindow => ({
  name: 'w',
  kind: 'Deny',
  dtstart: '20260824T090000',
  dtend: '20260824T170000',
  ...overrides
});

const at = (iso: string) => new Date(iso);

const parseRange = (windows: PromotionWindow[], from: Date, to: Date) =>
  parsePromotionWindows(windows, { range: { from, to } });

const spans = (occurrences: { start: Date; end: Date }[]) =>
  occurrences.map((o) => [o.start.toISOString(), o.end.toISOString()]);

const withTimeZone = <T>(timeZone: string, run: () => T): T => {
  const original = process.env.TZ;
  process.env.TZ = timeZone;
  try {
    return run();
  } finally {
    process.env.TZ = original;
  }
};

const timeZones = [
  'UTC',
  'Asia/Tokyo',
  'America/New_York',
  'Australia/Eucla',
  'Pacific/Kiritimati'
];

describe('parsePromotionWindows', () => {
  it('yields one occurrence for a window with no rrule', () => {
    expect(
      spans(parseRange([makeWindow({})], at('2026-08-01T00:00:00Z'), at('2026-09-01T00:00:00Z')))
    ).toEqual([['2026-08-24T09:00:00.000Z', '2026-08-24T17:00:00.000Z']]);
  });

  it('drops a window with no rrule that falls outside the range', () => {
    expect(
      parseRange([makeWindow({})], at('2026-09-01T00:00:00Z'), at('2026-10-01T00:00:00Z'))
    ).toHaveLength(0);
  });

  it('keeps a window with no rrule that is still running at the range start', () => {
    expect(
      spans(parseRange([makeWindow({})], at('2026-08-24T12:00:00Z'), at('2026-08-24T13:00:00Z')))
    ).toEqual([['2026-08-24T09:00:00.000Z', '2026-08-24T17:00:00.000Z']]);
  });

  it('expands a weekday rule across the range', () => {
    expect(
      parseRange(
        [makeWindow({ rrule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' })],
        at('2026-08-24T00:00:00Z'),
        at('2026-08-31T00:00:00Z')
      ).map((o) => o.start.toISOString())
    ).toEqual([
      '2026-08-24T09:00:00.000Z',
      '2026-08-25T09:00:00.000Z',
      '2026-08-26T09:00:00.000Z',
      '2026-08-27T09:00:00.000Z',
      '2026-08-28T09:00:00.000Z'
    ]);
  });

  it('expands a daily rule only within the range', () => {
    expect(
      parseRange(
        [makeWindow({ rrule: 'FREQ=DAILY' })],
        at('2026-08-24T00:00:00Z'),
        at('2026-08-27T00:00:00Z')
      )
    ).toHaveLength(3);
  });

  it('gives every occurrence the duration of dtend minus dtstart', () => {
    const occurrences = parseRange(
      [makeWindow({ dtend: '20260826T090000', rrule: 'FREQ=WEEKLY' })],
      at('2026-08-24T00:00:00Z'),
      at('2026-09-21T00:00:00Z')
    );

    expect(occurrences).toHaveLength(4);
    for (const occurrence of occurrences) {
      expect(occurrence.end.getTime() - occurrence.start.getTime()).toBe(2 * 86_400_000);
    }
  });

  it('keeps a recurring occurrence that is still running at the range start', () => {
    expect(
      spans(
        parseRange(
          [
            makeWindow({
              dtstart: '20260824T220000',
              dtend: '20260825T060000',
              rrule: 'FREQ=DAILY'
            })
          ],
          at('2026-08-25T00:00:00Z'),
          at('2026-08-25T12:00:00Z')
        )
      )
    ).toEqual([['2026-08-24T22:00:00.000Z', '2026-08-25T06:00:00.000Z']]);
  });

  it('drops a recurring occurrence that ends when the range starts', () => {
    expect(
      parseRange(
        [
          makeWindow({
            dtstart: '20260824T220000',
            dtend: '20260825T060000',
            rrule: 'FREQ=DAILY'
          })
        ],
        at('2026-08-25T06:00:00Z'),
        at('2026-08-25T12:00:00Z')
      )
    ).toHaveLength(0);
  });

  it('yields nothing when the rrule cannot be parsed', () => {
    expect(
      parseRange(
        [makeWindow({ rrule: 'FREQ=BOGUS' })],
        at('2026-08-01T00:00:00Z'),
        at('2026-09-01T00:00:00Z')
      )
    ).toHaveLength(0);
  });

  it.each([
    ['an unparseable dtstart', { dtstart: 'nonsense' }],
    ['a missing dtend', { dtend: undefined }],
    ['an empty dtstart', { dtstart: '' }]
  ])('yields nothing for a window with %s', (_label, overrides) => {
    expect(
      parseRange([makeWindow(overrides)], at('2026-08-01T00:00:00Z'), at('2026-09-01T00:00:00Z'))
    ).toHaveLength(0);
  });

  it('keeps the other windows when one cannot be parsed', () => {
    expect(
      parseRange(
        [makeWindow({ name: 'broken', dtstart: 'nonsense' }), makeWindow({ name: 'ok' })],
        at('2026-08-01T00:00:00Z'),
        at('2026-09-01T00:00:00Z')
      ).map((o) => o.name)
    ).toEqual(['ok']);
  });

  it('replaces the iCal fields and keeps everything else', () => {
    const [occurrence] = parseRange(
      [
        makeWindow({
          name: 'weekend-freeze',
          kind: 'Allow',
          stageSelector: { name: 'glob:prod-*' },
          projectSelector: { matchLabels: { env: 'prod' } }
        })
      ],
      at('2026-08-01T00:00:00Z'),
      at('2026-09-01T00:00:00Z')
    );

    expect(occurrence).toEqual({
      name: 'weekend-freeze',
      kind: 'Allow',
      stageSelector: { name: 'glob:prod-*' },
      projectSelector: { matchLabels: { env: 'prod' } },
      start: at('2026-08-24T09:00:00Z'),
      end: at('2026-08-24T17:00:00Z')
    });
  });

  it('flattens every window in the list', () => {
    expect(
      parseRange(
        [makeWindow({ name: 'a', rrule: 'FREQ=DAILY' }), makeWindow({ name: 'b' })],
        at('2026-08-24T00:00:00Z'),
        at('2026-08-27T00:00:00Z')
      ).map((o) => o.name)
    ).toEqual(['a', 'a', 'a', 'b']);
  });
});

describe('parsePromotionWindows across time zones', () => {
  const parseTzWindow = () =>
    spans(
      parseRange(
        [
          makeWindow({
            dtstart: 'TZID=America/New_York:20260101T090000',
            dtend: 'TZID=America/New_York:20260101T170000',
            rrule: 'FREQ=DAILY'
          })
        ],
        at('2026-01-01T00:00:00Z'),
        at('2026-01-04T00:00:00Z')
      )
    );

  const expectedSpans: Record<string, string[][]> = {
    UTC: [
      ['2026-01-01T14:00:00.000Z', '2026-01-01T22:00:00.000Z'],
      ['2026-01-02T14:00:00.000Z', '2026-01-02T22:00:00.000Z'],
      ['2026-01-03T14:00:00.000Z', '2026-01-03T22:00:00.000Z']
    ],
    'Asia/Tokyo': [
      ['2026-01-01T23:00:00.000Z', '2026-01-02T07:00:00.000Z'],
      ['2026-01-02T23:00:00.000Z', '2026-01-03T07:00:00.000Z'],
      ['2026-01-03T23:00:00.000Z', '2026-01-04T07:00:00.000Z']
    ],
    'America/New_York': [
      ['2026-01-01T09:00:00.000Z', '2026-01-01T17:00:00.000Z'],
      ['2026-01-02T09:00:00.000Z', '2026-01-02T17:00:00.000Z'],
      ['2026-01-03T09:00:00.000Z', '2026-01-03T17:00:00.000Z']
    ],
    'Australia/Eucla': [
      ['2026-01-01T22:45:00.000Z', '2026-01-02T06:45:00.000Z'],
      ['2026-01-02T22:45:00.000Z', '2026-01-03T06:45:00.000Z'],
      ['2026-01-03T22:45:00.000Z', '2026-01-04T06:45:00.000Z']
    ],
    'Pacific/Kiritimati': [
      ['2026-01-02T04:00:00.000Z', '2026-01-02T12:00:00.000Z'],
      ['2026-01-03T04:00:00.000Z', '2026-01-03T12:00:00.000Z']
    ]
  };

  it.each(timeZones)('resolves a New York window into the clock of a machine in %s', (tz) => {
    expect(withTimeZone(tz, parseTzWindow)).toEqual(expectedSpans[tz]);
  });

  it("follows the window zone's DST shift into the viewer's zone", () => {
    expect(
      withTimeZone('Asia/Tokyo', () =>
        parseRange(
          [
            makeWindow({
              dtstart: 'TZID=America/New_York:20260306T090000',
              dtend: 'TZID=America/New_York:20260306T100000',
              rrule: 'FREQ=DAILY'
            })
          ],
          at('2026-03-06T00:00:00Z'),
          at('2026-03-11T00:00:00Z')
        ).map((o) => o.start.toISOString())
      )
    ).toEqual([
      '2026-03-06T23:00:00.000Z',
      '2026-03-07T23:00:00.000Z',
      '2026-03-08T22:00:00.000Z',
      '2026-03-09T22:00:00.000Z',
      '2026-03-10T22:00:00.000Z'
    ]);
  });

  it('resolves two windows authored in different zones into the viewer clock', () => {
    expect(
      withTimeZone('Pacific/Kiritimati', () =>
        parseRange(
          [
            makeWindow({
              name: 'tokyo',
              dtstart: 'TZID=Asia/Tokyo:20260101T090000',
              dtend: 'TZID=Asia/Tokyo:20260101T100000'
            }),
            makeWindow({
              name: 'new-york',
              dtstart: 'TZID=America/New_York:20260101T090000',
              dtend: 'TZID=America/New_York:20260101T100000'
            })
          ],
          at('2026-01-01T00:00:00Z'),
          at('2026-01-03T00:00:00Z')
        ).map((o) => [o.name, o.start.toISOString()])
      )
    ).toEqual([
      ['tokyo', '2026-01-01T14:00:00.000Z'],
      ['new-york', '2026-01-02T04:00:00.000Z']
    ]);
  });

  it('places a window identically across a DST shift in the viewer zone', () => {
    const window = {
      dtstart: 'TZID=Asia/Tokyo:20260308T000000',
      dtend: 'TZID=Asia/Tokyo:20260308T010000'
    };

    const parse = (rrule?: string) =>
      parseRange(
        [makeWindow(rrule ? { ...window, rrule } : window)],
        at('2026-03-07T00:00:00Z'),
        at('2026-03-09T00:00:00Z')
      );

    expect(withTimeZone('America/New_York', () => spans(parse()))).toEqual([
      ['2026-03-07T10:00:00.000Z', '2026-03-07T11:00:00.000Z']
    ]);

    expect(withTimeZone('America/New_York', () => spans(parse()))).toEqual(
      withTimeZone('America/New_York', () => spans(parse('FREQ=DAILY;COUNT=1')))
    );
  });

  it.each(timeZones)('places a window identically with and without an rrule in %s', (tz) => {
    const window = {
      dtstart: 'TZID=America/New_York:20260101T090000',
      dtend: 'TZID=America/New_York:20260101T170000'
    };

    const parse = (rrule?: string) =>
      parseRange(
        [makeWindow(rrule ? { ...window, rrule } : window)],
        at('2026-01-01T00:00:00Z'),
        at('2026-01-03T00:00:00Z')
      );

    expect(withTimeZone(tz, () => spans(parse()))).toEqual(
      withTimeZone(tz, () => spans(parse('FREQ=DAILY;COUNT=1')))
    );
  });
});
