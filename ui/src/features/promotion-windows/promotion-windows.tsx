import {
  faChevronLeft,
  faChevronRight,
  faLock,
  faLockOpen,
  faPlus,
  faWandMagicSparkles
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex, Segmented, Tag, Tooltip, Typography } from 'antd';
import { addMonths, addWeeks, endOfWeek, startOfWeek } from 'date-fns';
import { useMemo, useState } from 'react';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import { disabledOccurrenceStyle, occurrenceColors } from './occurrence-colors';
import { CalendarView, PromotionCalendar } from './promotion-calendar';
import { promotionWindowFromRange } from './promotion-window-form';
import { PromotionWindowModal } from './promotion-window-modal';
import { promotionWindowRecipes } from './recipes';
import { useGetPromotionWindowOccurrences } from './use-get-promotion-window-occurrences';
import { useIsPromotionWindowOpen } from './use-is-promotion-window-open';

type PromotionWindowsProps = {
  scope: 'project' | 'cluster';
  promotionWindows: PromotionWindow[];
  onUpdate: (promotionWindows: PromotionWindow[]) => void;
};

export const PromotionWindows = ({ scope, promotionWindows, onUpdate }: PromotionWindowsProps) => {
  const [view, setView] = useState<CalendarView>('timeGridWeek');
  const [date, setDate] = useState(() => new Date());

  const promotionWindowNames = useMemo(
    () => promotionWindows?.map((w) => w.name),
    [promotionWindows]
  );

  const createModal = useModal();
  const createWindow = (promotionWindow?: PromotionWindow) =>
    createModal.show((p) => (
      <PromotionWindowModal
        {...p}
        scope={scope}
        promotionWindow={promotionWindow}
        names={promotionWindowNames}
        onSubmit={(promotionWindow) => onUpdate([...promotionWindows, promotionWindow])}
      />
    ));

  const confirm = useConfirmModal();

  const editModal = useModal();

  const onDelete = (promotionWindow: PromotionWindow) =>
    confirm({
      title: 'Delete Promotion Window',
      content: (
        <Typography.Text>
          Are you sure you want to delete promotion window{' '}
          <Typography.Text strong>{promotionWindow.name}</Typography.Text>?
        </Typography.Text>
      ),
      onOk: () => {
        onUpdate(promotionWindows.filter((window) => window.name !== promotionWindow.name));
        editModal.hide();
      }
    });

  const onEditModalShow = (promotionWindow: PromotionWindow) =>
    editModal.show((p) => (
      <PromotionWindowModal
        {...p}
        scope={scope}
        editing
        promotionWindow={promotionWindow}
        names={promotionWindowNames.filter((name) => name !== promotionWindow?.name)}
        onSubmit={(next) =>
          onUpdate(
            promotionWindows.map((window) => (window.name === promotionWindow.name ? next : window))
          )
        }
        onDelete={() => onDelete(promotionWindow)}
      />
    ));

  const isOpen = useIsPromotionWindowOpen(promotionWindows);

  const disabledCount = promotionWindows.filter(
    (promotionWindow) => promotionWindow.disabled
  ).length;

  const occurrences = useGetPromotionWindowOccurrences(promotionWindows, view, date);

  const shift = (direction: number) =>
    setDate(view === 'dayGridMonth' ? addMonths(date, direction) : addWeeks(date, direction));

  return (
    <Card
      title='Promotion Windows'
      type='inner'
      className='min-h-full'
      extra={
        <Dropdown.Button
          type='primary'
          icon={<FontAwesomeIcon icon={faWandMagicSparkles} size='sm' />}
          onClick={() => createWindow()}
          menu={{
            items: [
              {
                key: 'recipes',
                type: 'group',
                label: 'Start from a recipe',
                children: promotionWindowRecipes.map((recipe) => ({
                  key: recipe.key,
                  label: (
                    <Flex vertical gap={2} className='max-w-xs py-1'>
                      <Flex align='center' gap={6}>
                        <Typography.Text strong>{recipe.label}</Typography.Text>
                        <Tag
                          color={
                            recipe.kind === PromotionWindowKind.PromotionWindowKindAllow
                              ? 'green'
                              : 'red'
                          }
                          className='m-0 text-xs'
                        >
                          {recipe.kind}
                        </Tag>
                      </Flex>
                      <Typography.Text type='secondary' className='text-xs whitespace-normal'>
                        {recipe.description}
                      </Typography.Text>
                    </Flex>
                  ),
                  onClick: () => createWindow(recipe.create())
                }))
              }
            ]
          }}
        >
          <FontAwesomeIcon icon={faPlus} />
          New Window
        </Dropdown.Button>
      }
    >
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
          <Tooltip
            title={
              isOpen
                ? 'No window is currently blocking promotion. Projected from the windows on this calendar; Kargo evaluates each Stage against its own selectors.'
                : 'Projected from the windows on this calendar; Kargo evaluates each Stage against its own selectors.'
            }
          >
            <Tag color={isOpen ? 'green' : 'red'} className='m-0 cursor-help'>
              <FontAwesomeIcon icon={isOpen ? faLockOpen : faLock} className='mr-1' size='sm' />
              {isOpen ? 'Promotions open' : 'Promotions closed'}
            </Tag>
          </Tooltip>
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
        onSelectSlot={(range) => createWindow(promotionWindowFromRange(range.start, range.end))}
        onSelectOccurrence={(occurrence) => {
          const promotionWindow = promotionWindows.find(
            (window) => window.name === occurrence.name
          );
          if (promotionWindow) {
            onEditModalShow(promotionWindow);
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
    </Card>
  );
};
