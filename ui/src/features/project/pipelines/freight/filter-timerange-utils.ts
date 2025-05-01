import { sub } from 'date-fns';

export type timerangeTypes =
  | '1-minute'
  | '15-minutes'
  | '30-minutes'
  | '1-hour'
  | '12-hours'
  | '1-day'
  | '1-week'
  | '1-month'
  | 'all-time';

export const timerangeToDate = (timerange: timerangeTypes) => {
  const now = new Date();
  switch (timerange) {
    case '1-day':
      return sub(now, { days: 1 });
    case '1-hour':
      return sub(now, { hours: 1 });
    case '1-minute':
      return sub(now, { minutes: 1 });
    case '1-month':
      return sub(now, { months: 1 });
    case '1-week':
      return sub(now, { weeks: 1 });
    case '12-hours':
      return sub(now, { hours: 12 });
    case '15-minutes':
      return sub(now, { minutes: 15 });
    case '30-minutes':
      return sub(now, { minutes: 30 });
  }

  return new Date(0);
};

export const timerangeToLabel = (timerange: timerangeTypes) => {
  switch (timerange) {
    case '1-day':
      return '1 Day';
    case '1-hour':
      return '1 Hour';
    case '1-minute':
      return '1 Minute';
    case '1-month':
      return '1 Month';
    case '1-week':
      return '1 Week';
    case '12-hours':
      return '12 Hours';
    case '15-minutes':
      return '15 Minutes';
    case '30-minutes':
      return '30 Minutes';
    case 'all-time':
      return 'All time';
  }
};

export const timerangeOrderedOptions: timerangeTypes[] = [
  '15-minutes',
  '30-minutes',
  '1-hour',
  '12-hours',
  '1-day',
  '1-week',
  '1-month',
  'all-time'
];
