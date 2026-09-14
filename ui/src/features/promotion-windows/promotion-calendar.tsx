import FullCalendar, { CalendarRef } from '@fullcalendar/react';
import dayGridPlugin from '@fullcalendar/react/daygrid';
import interactionPlugin from '@fullcalendar/react/interaction';
import timeGridPlugin from '@fullcalendar/react/timegrid';
import { Flex, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { format, isSameDay } from 'date-fns';
import { useEffect, useRef } from 'react';

import { disabledOccurrenceStyle, occurrenceColors } from './occurrence-colors';
import { PromotionWindowOccurrence } from './types';
import { fromViewerClockDate } from './viewer-clock';

import '@fullcalendar/react/skeleton.css';

export type CalendarView = 'timeGridWeek' | 'dayGridMonth';

type PromotionCalendarProps = {
  view: CalendarView;
  date: Date;
  occurrences: PromotionWindowOccurrence[];
  onSelectSlot: (range: { start: Date; end: Date }) => void;
  onSelectOccurrence: (occurrence: PromotionWindowOccurrence) => void;
};

export const PromotionCalendar = ({
  view,
  date,
  occurrences,
  onSelectSlot,
  onSelectOccurrence
}: PromotionCalendarProps) => {
  const calendar = useRef<CalendarRef>(null);

  useEffect(() => {
    calendar.current?.getApi().changeView(view, date);
  }, [view, date]);

  return (
    <FullCalendar
      ref={calendar}
      plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
      initialView={view}
      initialDate={date}
      headerToolbar={false}
      height='clamp(400px, calc(100vh - 300px), 760px)'
      nowIndicator
      dayMaxEvents
      selectable
      expandRows
      slotDuration='00:30:00'
      slotMinHeight={26}
      slotEventOverlap={false}
      slotHeaderInterval='01:00:00'
      scrollTime='07:00:00'
      slotHeaderFormat={{ hour: '2-digit', minute: '2-digit', hour12: false }}
      eventTimeFormat={{ hour: '2-digit', minute: '2-digit', hour12: false }}
      events={occurrences.map((occurrence) => {
        const start = fromViewerClockDate(occurrence.start);
        const end = fromViewerClockDate(occurrence.end);

        return {
          title: occurrence.name,
          start,
          end,
          allDay: end.getTime() - start.getTime() >= 86_400_000,
          extendedProps: { occurrence }
        };
      })}
      select={(info) => onSelectSlot({ start: info.start, end: info.end })}
      eventClick={(info) =>
        onSelectOccurrence(info.event.extendedProps.occurrence as PromotionWindowOccurrence)
      }
      viewClass={classNames(
        'overflow-hidden rounded-lg',
        'dark:bg-neutral-900 dark:text-neutral-100',
        'border border-solid border-gray-200 dark:border-neutral-700'
      )}
      tableHeaderClass='dark:bg-neutral-900'
      dayHeaderRowClass='border border-solid border-gray-200 dark:border-neutral-700'
      dayHeaderClass='justify-center'
      dayHeaderInnerClass='mx-1 my-1.5'
      dayHeaderContent={(info) =>
        view === 'dayGridMonth' ? (
          <Typography.Text strong type='secondary' className='text-[10px] uppercase tracking-wider'>
            {format(info.date, 'EEE')}
          </Typography.Text>
        ) : (
          <>
            <Typography.Text
              strong
              type={info.isToday ? undefined : 'secondary'}
              className={classNames(
                'text-[10px] uppercase tracking-wider',
                info.isToday && '!text-blue-600 dark:!text-blue-400'
              )}
            >
              {format(info.date, 'EEE')}
            </Typography.Text>
            <Typography.Text
              strong={info.isToday}
              className={classNames(
                'grid h-6 min-w-6 place-items-center rounded-full px-1.5 text-sm tabular-nums',
                info.isToday ? 'bg-blue-600 !text-white' : 'font-medium'
              )}
            >
              {format(info.date, 'd')}
            </Typography.Text>
          </>
        )
      }
      dayRowClass='border border-solid border-gray-200 dark:border-neutral-700'
      dayCellClass={(info) =>
        classNames(
          'border border-solid border-gray-200 dark:border-neutral-700',
          info.isToday && 'bg-blue-50/70 dark:bg-blue-500/10',
          info.isOther && 'bg-gray-50/70 dark:bg-neutral-800/40'
        )
      }
      dayCellTopClass='flex flex-row justify-start'
      dayCellTopInnerClass='mx-2 my-1'
      dayCellTopContent={(info) => (
        <Typography.Text
          type={!info.isToday && info.isOther ? 'secondary' : undefined}
          className={classNames(
            'grid h-5 min-w-5 place-items-center rounded-full text-[11px] tabular-nums',
            info.isToday ? 'bg-blue-600 font-semibold !text-white' : !info.isOther && 'font-medium'
          )}
        >
          {info.text}
        </Typography.Text>
      )}
      dayLaneClass={(info) =>
        classNames(
          'border border-solid border-gray-200 dark:border-neutral-700',
          info.isToday && 'bg-blue-50/60 dark:bg-blue-500/10'
        )
      }
      slotLaneClass={(info) =>
        classNames(
          'border border-solid border-gray-100 dark:border-neutral-800',
          info.isMinor && 'border-dotted'
        )
      }
      slotHeaderInnerClass={classNames(
        'mx-1 my-0.5 text-[10px] font-medium tabular-nums',
        'text-gray-400 dark:text-neutral-500'
      )}
      slotHeaderDividerClass='border-solid border-0 border-r border-gray-200 dark:border-neutral-700'
      allDayHeaderContent={() => null}
      allDayDividerClass='border border-solid border-gray-200 dark:border-neutral-700'
      nowIndicatorLineClass='-mt-px border-solid border-0 border-t-2 border-red-500'
      nowIndicatorDotClass='-ms-1 -mt-1 size-2 rounded-full bg-red-500'
      highlightClass='rounded border border-dashed border-blue-400 bg-blue-500/10'
      eventClass={(info) => {
        const occurrence = info.event.extendedProps.occurrence as PromotionWindowOccurrence;

        return classNames(
          'cursor-pointer overflow-hidden rounded border text-xs transition-colors',
          'mb-px shadow-[0_0_0_1px_#fff] dark:shadow-[0_0_0_1px_#171717]',
          occurrence.disabled ? 'border-dashed' : 'border-solid',
          occurrenceColors(occurrence.kind).event
        );
      }}
      eventContent={(info) => {
        const { start, end } = info.event;
        const span =
          start && end
            ? isSameDay(start, end)
              ? `${format(start, 'HH:mm')} - ${format(end, 'HH:mm')}`
              : `${format(start, 'MMM d, HH:mm')} - ${format(end, 'MMM d, HH:mm')}`
            : info.timeText;
        const occurrence = info.event.extendedProps.occurrence as PromotionWindowOccurrence;

        return (
          <Tooltip
            placement='top'
            title={
              <Flex vertical gap={2} className='max-w-xs'>
                <Typography.Text strong className='!text-inherit'>
                  {occurrence.name} ({occurrence.kind})
                </Typography.Text>
                {occurrence.disabled && (
                  <Typography.Text className='text-xs !text-inherit'>
                    Disabled - ignored when deciding whether promotions may run.
                  </Typography.Text>
                )}
                <Typography.Text className='text-xs tabular-nums opacity-75 !text-inherit'>
                  {span}
                </Typography.Text>
                {occurrence.description && (
                  <Typography.Text className='mt-1 text-xs whitespace-normal !text-inherit'>
                    {occurrence.description}
                  </Typography.Text>
                )}
              </Flex>
            }
          >
            <Flex
              className='h-full w-full min-w-0 px-1.5 py-0.5'
              style={occurrence.disabled ? disabledOccurrenceStyle : undefined}
              vertical={!info.isShort}
              align={info.isShort ? 'baseline' : undefined}
              gap={info.isShort ? 6 : undefined}
            >
              <Flex className='min-w-0' align='baseline' gap={6}>
                <span
                  className={classNames(
                    'size-1.5 shrink-0 translate-y-[-1px] rounded-full',
                    occurrenceColors(occurrence.kind).dot
                  )}
                />
                <Typography.Text
                  strong
                  delete={occurrence.disabled}
                  className='truncate !text-inherit'
                >
                  {occurrence.name}
                </Typography.Text>
              </Flex>
              <Typography.Text className='shrink-0 truncate text-[10px] tabular-nums opacity-75 !text-inherit'>
                {span}
              </Typography.Text>
            </Flex>
          </Tooltip>
        );
      }}
      moreLinkClass={classNames(
        'rounded px-1 text-[11px] font-medium hover:bg-gray-100 dark:hover:bg-neutral-800',
        'text-gray-400 dark:text-neutral-500'
      )}
      popoverClass={classNames(
        'overflow-hidden rounded-lg bg-white shadow-lg dark:bg-neutral-800',
        'border border-solid border-gray-200 dark:border-neutral-700'
      )}
    />
  );
};
