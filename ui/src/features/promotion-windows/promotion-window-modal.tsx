import { faBan, faCircleCheck, faGlobe, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Button,
  Card,
  Checkbox,
  Flex,
  Input,
  Modal,
  Segmented,
  Select,
  Switch,
  Typography
} from 'antd';
import { addHours, isSameDay, startOfHour } from 'date-fns';
import { useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { RRule } from 'rrule';

import { DatePicker, TimePicker } from '@ui/features/common/date-picker';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import {
  combine,
  formValuesFromPromotionWindow,
  promotionWindowFromFormValues,
  promotionWindowFromRange,
  promotionWindowFormSchema,
  PromotionWindowFormValues
} from './promotion-window-form-utils';
import { RecurrenceFields } from './recurrence-fields';
import { SelectorFields } from './selector-fields';

type PromotionWindowModalProps = ModalComponentProps & {
  scope: 'project' | 'cluster';
  names: string[];
  editing?: boolean;
  promotionWindow?: PromotionWindow;
  onSubmit: (promotionWindow: PromotionWindow) => void;
  onDelete?: () => void;
};

export const PromotionWindowModal = ({
  visible,
  hide,
  editing,
  promotionWindow,
  scope,
  names,
  onSubmit,
  onDelete
}: PromotionWindowModalProps) => {
  const form = useForm<PromotionWindowFormValues>({
    defaultValues: formValuesFromPromotionWindow(
      promotionWindow ?? promotionWindowFromRange(new Date(), addHours(startOfHour(new Date()), 1))
    ),
    resolver: zodResolver(promotionWindowFormSchema)
  });

  const startDate = form.watch('startDate');
  const endDate = form.watch('endDate');
  const rrule = form.watch('rrule');
  const name = form.watch('name');
  const timeZone = form.watch('timeZone');

  const nameTaken = names.includes(name.trim());

  const [spansMultipleDays, setSpansMultipleDays] = useState(() => !isSameDay(endDate, startDate));
  const [editingTimeZone, setEditingTimeZone] = useState(false);

  const setEndTo = (date: Date) =>
    form.setValue('endDate', combine(date, form.getValues('endDate')), { shouldValidate: true });

  let summary = 'Does not repeat';
  try {
    summary = rrule ? new RRule(rrule).toText() : summary;
  } catch {
    summary = 'Custom recurrence rule';
  }

  const handleSubmit = form.handleSubmit((values) => {
    onSubmit(promotionWindowFromFormValues(values, scope));
    hide();
  });

  return (
    <FormProvider {...form}>
      <Modal
        open={visible}
        onCancel={hide}
        width={760}
        styles={{ body: { maxHeight: 'calc(100vh - 260px)', overflowY: 'auto' } }}
        title={editing ? 'Edit promotion window' : 'New promotion window'}
        okText={editing ? 'Save changes' : 'Create'}
        onOk={handleSubmit}
        okButtonProps={{ disabled: nameTaken }}
        footer={(_, { OkBtn, CancelBtn }) => (
          <Flex align='center' justify='space-between'>
            {onDelete ? (
              <Button danger icon={<FontAwesomeIcon icon={faTrash} size='sm' />} onClick={onDelete}>
                Delete
              </Button>
            ) : (
              <span />
            )}
            <Flex gap={8}>
              <CancelBtn />
              <OkBtn />
            </Flex>
          </Flex>
        )}
      >
        <FieldContainer control={form.control} name='name' label='Name' required>
          {({ field }) => (
            <>
              <Input {...field} placeholder='weekend-freeze' />
              {nameTaken && (
                <Typography.Text type='danger' className='text-xs'>
                  Another promotion window already uses that name
                </Typography.Text>
              )}
            </>
          )}
        </FieldContainer>

        <Flex gap={16} align='end'>
          <Flex vertical flex='0 0 220px'>
            <FieldContainer control={form.control} name='disabled'>
              {({ field }) => (
                <Card size='small'>
                  <Flex align='center' justify='space-between' gap={12}>
                    <Typography.Text strong>Enabled</Typography.Text>
                    <Switch
                      checked={!field.value}
                      onChange={(checked) => field.onChange(!checked)}
                    />
                  </Flex>
                </Card>
              )}
            </FieldContainer>
          </Flex>

          <Flex vertical flex={1} style={{ minWidth: 0 }}>
            <FieldContainer control={form.control} name='kind' label='Effect'>
              {({ field }) => (
                <Segmented
                  {...field}
                  block
                  options={[
                    {
                      value: PromotionWindowKind.PromotionWindowKindDeny,
                      label: (
                        <span>
                          <FontAwesomeIcon icon={faBan} className='mr-2' />
                          Deny
                        </span>
                      )
                    },
                    {
                      value: PromotionWindowKind.PromotionWindowKindAllow,
                      label: (
                        <span>
                          <FontAwesomeIcon icon={faCircleCheck} className='mr-2' />
                          Allow
                        </span>
                      )
                    }
                  ]}
                />
              )}
            </FieldContainer>
          </Flex>
        </Flex>

        <FieldContainer
          control={form.control}
          name='description'
          label='Description'
          tooltip='Shown when hovering the window on the calendar. Use it for the reason, owner or any clarification.'
        >
          {({ field }) => (
            <Input.TextArea
              {...field}
              rows={2}
              maxLength={1024}
              showCount
              placeholder='No promotions during the weekend release freeze'
            />
          )}
        </FieldContainer>

        <Flex gap={16}>
          <Flex vertical flex={1} style={{ minWidth: 0 }}>
            <FieldContainer control={form.control} name='startDate' label='Starts'>
              {({ field }) => (
                <Flex gap={8}>
                  <DatePicker
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={(date) => {
                      field.onChange(combine(date, field.value));
                      if (!spansMultipleDays) {
                        setEndTo(date);
                      }
                    }}
                    allowClear={false}
                    className='flex-1'
                    format='EEEE, MMM d, yyyy'
                  />
                  <TimePicker
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={(time) => field.onChange(combine(field.value, time))}
                    format='HH:mm'
                    minuteStep={15}
                    allowClear={false}
                    style={{ width: 108 }}
                  />
                </Flex>
              )}
            </FieldContainer>
          </Flex>

          <Flex vertical flex={1} style={{ minWidth: 0 }}>
            <FieldContainer control={form.control} name='endDate' label='Ends'>
              {({ field }) => (
                <Flex gap={8}>
                  {spansMultipleDays && (
                    <DatePicker
                      value={field.value}
                      onBlur={field.onBlur}
                      onChange={(date) => field.onChange(combine(date, field.value))}
                      allowClear={false}
                      className='flex-1'
                      minDate={startDate}
                      format='EEEE, MMM d, yyyy'
                    />
                  )}
                  <TimePicker
                    value={field.value}
                    onBlur={field.onBlur}
                    onChange={(time) => field.onChange(combine(field.value, time))}
                    format='HH:mm'
                    minuteStep={15}
                    allowClear={false}
                    style={{ width: 108 }}
                  />
                </Flex>
              )}
            </FieldContainer>
          </Flex>
        </Flex>

        <div className='mb-6'>
          <Flex align='center' justify='space-between' gap={8} wrap>
            <Checkbox
              checked={spansMultipleDays}
              onChange={(event) => {
                setSpansMultipleDays(event.target.checked);
                if (!event.target.checked) {
                  setEndTo(startDate);
                }
              }}
            >
              Spans more than one day
            </Checkbox>
            <Button
              type='link'
              size='small'
              icon={<FontAwesomeIcon icon={faGlobe} size='sm' />}
              onClick={() => setEditingTimeZone(!editingTimeZone)}
            >
              Time zone: {timeZone}
            </Button>
          </Flex>

          {editingTimeZone && (
            <FieldContainer
              control={form.control}
              name='timeZone'
              className='mt-3'
              formItemClassName='!mb-0'
            >
              {({ field }) => (
                <Select
                  {...field}
                  showSearch
                  options={['UTC', ...Intl.supportedValuesOf('timeZone')].map((zone) => ({
                    value: zone,
                    label: zone
                  }))}
                />
              )}
            </FieldContainer>
          )}
        </div>

        <FieldContainer control={form.control} name='rrule' label='Repeats' description={summary}>
          {({ field }) => (
            <RecurrenceFields value={field.value} onChange={field.onChange} startDate={startDate} />
          )}
        </FieldContainer>

        {scope === 'cluster' ? (
          <Flex gap={16} align='start'>
            <Flex vertical flex={1} style={{ minWidth: 0 }}>
              <SelectorFields subject='project' />
            </Flex>
            <Flex vertical flex={1} style={{ minWidth: 0 }}>
              <SelectorFields subject='stage' />
            </Flex>
          </Flex>
        ) : (
          <SelectorFields subject='stage' />
        )}
      </Modal>
    </FormProvider>
  );
};
