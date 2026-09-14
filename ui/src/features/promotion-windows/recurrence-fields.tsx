import { Card, Checkbox, Flex, InputNumber, Radio, Select, Typography } from 'antd';
import { addYears, endOfDay, format, getDate, getDay } from 'date-fns';
import { Frequency, Options, RRule, Weekday } from 'rrule';

import { DatePicker } from '@ui/features/common/date-picker';

const asArray = <T,>(value: T | T[] | null | undefined): T[] =>
  value === null || value === undefined ? [] : Array.isArray(value) ? value : [value];

const FREQUENCY_OPTIONS = [
  { label: 'Does not repeat', value: 'none' as const },
  { label: 'Hourly', value: Frequency.HOURLY },
  { label: 'Daily', value: Frequency.DAILY },
  { label: 'Weekly', value: Frequency.WEEKLY },
  { label: 'Monthly', value: Frequency.MONTHLY },
  { label: 'Yearly', value: Frequency.YEARLY }
];

const INTERVAL_UNITS: Record<number, string> = {
  [Frequency.HOURLY]: 'hour(s)',
  [Frequency.DAILY]: 'day(s)',
  [Frequency.WEEKLY]: 'week(s)',
  [Frequency.MONTHLY]: 'month(s)',
  [Frequency.YEARLY]: 'year(s)'
};

const WEEKDAY_OPTIONS = [RRule.MO, RRule.TU, RRule.WE, RRule.TH, RRule.FR, RRule.SA, RRule.SU].map(
  (weekday) => ({ label: weekday.toString(), value: weekday.weekday })
);

const ORDINALS = ['first', 'second', 'third', 'fourth', 'fifth'];

type RecurrenceFieldsProps = {
  value: Partial<Options> | null;
  onChange: (value: Partial<Options> | null) => void;
  startDate: Date;
};

export const RecurrenceFields = ({ value, onChange, startDate }: RecurrenceFieldsProps) => {
  const patch = (next: Partial<Options>) => onChange({ ...value, ...next });

  const weekdays = asArray(value?.byweekday).map((day) =>
    day instanceof Weekday ? day.weekday : Number(day)
  );
  const monthlyMode = asArray(value?.bymonthday).includes(-1)
    ? 'lastDay'
    : weekdays.length
      ? 'nthWeekday'
      : 'dayOfMonth';
  const ends = value?.until ? 'until' : value?.count ? 'count' : 'never';

  const nth = Math.ceil(getDate(startDate) / 7);

  return (
    <>
      <Select<'none' | Frequency>
        className='w-full'
        value={value?.freq ?? 'none'}
        onChange={(freq) => onChange(freq === 'none' ? null : { freq })}
        options={FREQUENCY_OPTIONS}
      />

      {value?.freq !== undefined && (
        <Card size='small' className='mt-3'>
          <Flex gap={8} align='center'>
            <Typography.Text>Every</Typography.Text>
            <InputNumber
              min={1}
              max={999}
              style={{ width: 72 }}
              value={value.interval ?? 1}
              onChange={(interval) =>
                patch({ interval: interval && interval > 1 ? interval : undefined })
              }
            />
            <Typography.Text>{INTERVAL_UNITS[value.freq]}</Typography.Text>
          </Flex>

          {value.freq === Frequency.WEEKLY && (
            <Checkbox.Group
              className='mt-3'
              options={WEEKDAY_OPTIONS}
              value={weekdays}
              onChange={(days) => patch({ byweekday: days })}
            />
          )}

          {value.freq === Frequency.MONTHLY && (
            <Radio.Group
              className='mt-3'
              value={monthlyMode}
              onChange={(event) =>
                patch(
                  event.target.value === 'lastDay'
                    ? { bymonthday: [-1], byweekday: null }
                    : event.target.value === 'nthWeekday'
                      ? {
                          bymonthday: null,
                          byweekday: [new Weekday((getDay(startDate) + 6) % 7, nth)]
                        }
                      : { bymonthday: [getDate(startDate)], byweekday: null }
                )
              }
              options={[
                { label: `On day ${getDate(startDate)}`, value: 'dayOfMonth' },
                {
                  label: `On the ${ORDINALS[nth - 1]} ${format(startDate, 'EEEE')}`,
                  value: 'nthWeekday'
                },
                { label: 'On the last day', value: 'lastDay' }
              ]}
            />
          )}

          <Flex gap={8} align='center' className='mt-3'>
            <Radio.Group
              value={ends}
              onChange={(event) =>
                patch(
                  event.target.value === 'until'
                    ? { until: endOfDay(addYears(startDate, 1)), count: null }
                    : event.target.value === 'count'
                      ? { until: null, count: 10 }
                      : { until: null, count: null }
                )
              }
              options={[
                { label: 'Forever', value: 'never' },
                { label: 'Until', value: 'until' },
                { label: 'After', value: 'count' }
              ]}
            />

            {ends === 'until' && (
              <DatePicker
                allowClear={false}
                value={value.until}
                onChange={(date) => patch({ until: endOfDay(date) })}
              />
            )}

            {ends === 'count' && (
              <InputNumber
                min={1}
                max={999}
                style={{ width: 132 }}
                addonAfter='times'
                value={value.count}
                onChange={(count) => patch({ count: count ?? 1 })}
              />
            )}
          </Flex>
        </Card>
      )}
    </>
  );
};
