import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Segmented, Typography } from 'antd';
import { addMonths, addWeeks, endOfWeek, startOfWeek } from 'date-fns';
import { useState } from 'react';

import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import { disabledOccurrenceStyle, occurrenceColors } from './occurrence-colors';
import { CalendarView, PromotionCalendar } from './promotion-calendar';
import { promotionWindowFromRange } from './promotion-window-form-utils';
import { useGetPromotionWindowOccurrences } from './use-get-promotion-window-occurrences';

type PromotionWindowsCalendarViewProps = {
  promotionWindows: PromotionWindow[];
  onCreate: (promotionWindow: PromotionWindow) => void;
  onEdit: (promotionWindow: PromotionWindow) => void;
};

export const PromotionWindowsCalendarView = ({
  promotionWindows,
  onCreate,
  onEdit
}: PromotionWindowsCalendarViewProps) => {
  const [view, setView] = useState<CalendarView>('timeGridWeek');
  const [date, setDate] = useState(() => new Date());

  const occurrences = useGetPromotionWindowOccurrences(promotionWindows, view, date);

  const disabledCount = promotionWindows.filter(
    (promotionWindow) => promotionWindow.disabled
  ).length;

  const shift = (direction: number) =>
    setDate(view === 'dayGridMonth' ? addMonths(date, direction) : addWeeks(date, direction));

  return (
    <>
      <Flex align='center' justify='space-between' className='mb-3' gap={8} wrap>
        <Flex align='center' gap={8}>
          <Button size='small' onClick={() => setDate(new Date())}>
            Today
          </Button>
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faChevronLeft} size='sm' />}
            onClick={() => shift(-1)}
          />
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faChevronRight} size='sm' />}
            onClick={() => shift(1)}
          />
          <Typography.Text strong>
            {view === 'dayGridMonth'
              ? date.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })
              : `${startOfWeek(date).toLocaleDateString(undefined, {
                  month: 'short',
                  day: 'numeric'
                })} - ${endOfWeek(date).toLocaleDateString(undefined, {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric'
                })}`}
          </Typography.Text>
        </Flex>
        <Segmented
          size='small'
          value={view}
          onChange={setView}
          options={[
            { label: 'Week', value: 'timeGridWeek' },
            { label: 'Month', value: 'dayGridMonth' }
          ]}
        />
      </Flex>

      <PromotionCalendar
        view={view}
        date={date}
        occurrences={occurrences}
        onSelectSlot={(range) => onCreate(promotionWindowFromRange(range.start, range.end))}
        onSelectOccurrence={(occurrence) => {
          const promotionWindow = promotionWindows.find(
            (window) => window.name === occurrence.name
          );
          if (promotionWindow) {
            onEdit(promotionWindow);
          }
        }}
      />

      <Flex justify='space-between' align='center' className='mt-2' gap={8} wrap>
        <Flex gap={12} align='center'>
          {[
            [PromotionWindowKind.PromotionWindowKindAllow, 'Allow'],
            [PromotionWindowKind.PromotionWindowKindDeny, 'Deny']
          ].map(([kind, label]) => (
            <Flex key={label} align='center' gap={6}>
              <span
                className={`size-2 rounded-full ${occurrenceColors(kind as PromotionWindowKind).dot}`}
              />
              <Typography.Text type='secondary' className='text-xs'>
                {label}
              </Typography.Text>
            </Flex>
          ))}
          {disabledCount > 0 && (
            <Flex align='center' gap={6}>
              <span
                className='size-2.5 rounded-xs border border-dashed border-gray-400 dark:border-neutral-500'
                style={disabledOccurrenceStyle}
              />
              <Typography.Text type='secondary' className='text-xs'>
                Disabled
              </Typography.Text>
            </Flex>
          )}
        </Flex>
        <Typography.Text type='secondary' className='text-xs'>
          {promotionWindows.length
            ? `${promotionWindows.length} window${promotionWindows.length === 1 ? '' : 's'} configured${disabledCount ? ` (${disabledCount} disabled)` : ''}. Drag on the grid to draft one.`
            : 'No windows yet - promotions are unconstrained. Drag on the grid to draft one.'}
        </Typography.Text>
      </Flex>
    </>
  );
};
