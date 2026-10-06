import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Typography } from 'antd';
import { addHours, addMonths, startOfHour } from 'date-fns';
import { useState } from 'react';

import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import { disabledOccurrenceStyle, occurrenceColors } from './occurrence-colors';
import { PromotionCalendar } from './promotion-calendar';
import { combine, promotionWindowFromRange } from './promotion-window-form-utils';
import { useGetPromotionWindowOccurrences } from './use-get-promotion-window-occurrences';

type PromotionWindowsCalendarViewProps = {
  promotionWindows: PromotionWindow[];
  onCreate: (promotionWindow: PromotionWindow) => void;
  onEdit: (promotionWindow: PromotionWindow) => void;
};

/** A one hour draft on `day`, starting at the hour the viewer is currently in. */
const draftFromDay = (day: Date) => {
  const start = combine(day, startOfHour(new Date()));

  return promotionWindowFromRange(start, addHours(start, 1));
};

export const PromotionWindowsCalendarView = ({
  promotionWindows,
  onCreate,
  onEdit
}: PromotionWindowsCalendarViewProps) => {
  const [date, setDate] = useState(() => new Date());

  const occurrences = useGetPromotionWindowOccurrences(promotionWindows, date);

  const disabledCount = promotionWindows.filter(
    (promotionWindow) => promotionWindow.disabled
  ).length;

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
            onClick={() => setDate(addMonths(date, -1))}
          />
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faChevronRight} size='sm' />}
            onClick={() => setDate(addMonths(date, 1))}
          />
          <Typography.Text strong>
            {date.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}
          </Typography.Text>
        </Flex>
      </Flex>

      <PromotionCalendar
        date={date}
        occurrences={occurrences}
        onSelectDay={(day) => {
          setDate(day);
          onCreate(draftFromDay(day));
        }}
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
            ? `${promotionWindows.length} window${promotionWindows.length === 1 ? '' : 's'} configured${disabledCount ? ` (${disabledCount} disabled)` : ''}. Click a day to draft one.`
            : 'No windows yet - promotions are unconstrained. Click a day to draft one.'}
        </Typography.Text>
      </Flex>
    </>
  );
};
