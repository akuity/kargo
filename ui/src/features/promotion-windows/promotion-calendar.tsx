import { Flex, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { endOfDay, format, isSameDay, startOfDay } from 'date-fns';
import { useMemo } from 'react';

import { Calendar } from '@ui/features/common/calendar';
import { selectorLines } from '@ui/features/common/selector/selector-utils';

import { disabledOccurrenceStyle, occurrenceColors } from './occurrence-colors';
import { PromotionWindowOccurrence } from './types';
import { fromViewerClockDate } from './viewer-clock';

/**
 * An occurrence paired with the real instants it renders at. Occurrences carry
 * viewer-clock dates, which must be converted before they can be compared with
 * the calendar's own (local) dates.
 */
type CalendarOccurrence = {
  occurrence: PromotionWindowOccurrence;
  start: Date;
  end: Date;
};

const selectedDayBg =
  "[[data-theme='light']_&_.ant-picker-cell-selected_.ant-picker-calendar-date]:!bg-[#e3ebf7]";

type PromotionCalendarProps = {
  date: Date;
  occurrences: PromotionWindowOccurrence[];
  onSelectDay: (day: Date) => void;
  onSelectOccurrence: (occurrence: PromotionWindowOccurrence) => void;
};

const fullSpan = ({ start, end }: CalendarOccurrence) =>
  isSameDay(start, end)
    ? `${format(start, 'HH:mm')} - ${format(end, 'HH:mm')}`
    : `${format(start, 'MMM d, HH:mm')} - ${format(end, 'MMM d, HH:mm')}`;

/** The portion of an occurrence that falls on a single day. */
const daySpan = ({ start, end }: CalendarOccurrence, day: Date) => {
  const startsToday = isSameDay(start, day);
  const endsToday = isSameDay(end, day);

  if (startsToday && endsToday) {
    return `${format(start, 'HH:mm')} - ${format(end, 'HH:mm')}`;
  }
  if (startsToday) {
    return `${format(start, 'HH:mm')} →`;
  }
  if (endsToday) {
    return `→ ${format(end, 'HH:mm')}`;
  }
  return 'All day';
};

export const PromotionCalendar = ({
  date,
  occurrences,
  onSelectDay,
  onSelectOccurrence
}: PromotionCalendarProps) => {
  const calendarOccurrences = useMemo<CalendarOccurrence[]>(
    () =>
      occurrences
        .map((occurrence) => ({
          occurrence,
          start: fromViewerClockDate(occurrence.start),
          end: fromViewerClockDate(occurrence.end)
        }))
        .sort((a, b) => a.start.getTime() - b.start.getTime()),
    [occurrences]
  );

  return (
    <Calendar
      value={date}
      mode='month'
      headerRender={() => null}
      onSelect={(day, info) => {
        if (info.source === 'date') {
          onSelectDay(day);
        }
      }}
      className={classNames(
        'overflow-hidden rounded-lg px-3',
        'border border-solid border-gray-200 dark:border-neutral-700',
        selectedDayBg
      )}
      cellRender={(day, info) => {
        if (info.type !== 'date') {
          return null;
        }

        const dayStart = startOfDay(day);
        const dayEnd = endOfDay(day);
        const dayOccurrences = calendarOccurrences.filter(
          ({ start, end }) => start <= dayEnd && end >= dayStart
        );

        if (!dayOccurrences.length) {
          return null;
        }

        return (
          <Flex vertical gap={2} className='min-w-0'>
            {dayOccurrences.map(({ occurrence, start, end }) => (
              <Tooltip
                key={`${occurrence.name}-${start.toISOString()}`}
                placement='top'
                title={
                  <Flex vertical gap={2} className='max-w-xs'>
                    <Typography.Text strong className='line-clamp-2 break-words !text-inherit'>
                      {occurrence.name} ({occurrence.kind})
                    </Typography.Text>
                    {occurrence.disabled && (
                      <Typography.Text className='text-xs !text-inherit'>
                        Disabled - ignored when deciding whether promotions may run.
                      </Typography.Text>
                    )}
                    <Typography.Text className='text-xs tabular-nums opacity-75 !text-inherit'>
                      {fullSpan({ occurrence, start, end })}
                    </Typography.Text>
                    {(
                      [
                        ['Stages', occurrence.stageSelector],
                        ['Projects', occurrence.projectSelector]
                      ] as const
                    ).map(([label, selector]) => {
                      const lines = selectorLines(selector);

                      return (
                        lines.length > 0 && (
                          <Flex key={label} gap={6} className='min-w-0 text-xs'>
                            <Typography.Text className='shrink-0 opacity-60 !text-inherit'>
                              {label}
                            </Typography.Text>
                            <Typography.Text className='line-clamp-2 break-words !text-inherit'>
                              {lines.join(', ')}
                            </Typography.Text>
                          </Flex>
                        )
                      );
                    })}
                    {occurrence.description && (
                      <Typography.Text
                        className={classNames(
                          'mt-1 text-xs whitespace-normal !text-inherit',
                          'line-clamp-4 break-words'
                        )}
                      >
                        {occurrence.description}
                      </Typography.Text>
                    )}
                  </Flex>
                }
              >
                <Flex
                  align='center'
                  gap={6}
                  className={classNames(
                    'min-w-0 cursor-pointer rounded border px-1.5 text-xs transition-colors',
                    occurrence.disabled ? 'border-dashed' : 'border-solid',
                    occurrenceColors(occurrence.kind).event
                  )}
                  style={occurrence.disabled ? disabledOccurrenceStyle : undefined}
                  onClick={(event) => {
                    event.stopPropagation();
                    onSelectOccurrence(occurrence);
                  }}
                >
                  <Typography.Text
                    strong
                    delete={occurrence.disabled}
                    className='min-w-0 truncate !text-inherit'
                  >
                    {occurrence.name}
                  </Typography.Text>
                  <Typography.Text
                    className={classNames(
                      'ms-auto shrink-0 text-[10px] tabular-nums opacity-75',
                      '!text-inherit'
                    )}
                  >
                    {daySpan({ occurrence, start, end }, day)}
                  </Typography.Text>
                </Flex>
              </Tooltip>
            ))}
          </Flex>
        );
      }}
    />
  );
};
