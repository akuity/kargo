import { sub } from 'date-fns';

export type timerangeTypes = '1-hour' | '1-day' | '1-week' | '1-month' | 'all-time';

export const timerangeToDate = (timerange: timerangeTypes) => {
  const now = new Date();
  switch (timerange) {
    case '1-day':
      return sub(now, { days: 1 });
    case '1-hour':
      return sub(now, { hours: 1 });
    case '1-month':
      return sub(now, { months: 1 });
    case '1-week':
      return sub(now, { weeks: 1 });
  }

  return new Date(0);
};

export const timerangeToLabel = (timerange: timerangeTypes) => {
  switch (timerange) {
    case '1-day':
      return '1 Day';
    case '1-hour':
      return '1 Hour';
    case '1-month':
      return '1 Month';
    case '1-week':
      return '1 Week';
    case 'all-time':
      return 'All time';
  }
};

export const timerangeOrderedOptions: timerangeTypes[] = [
  '1-hour',
  '1-day',
  '1-week',
  '1-month',
  'all-time'
];
