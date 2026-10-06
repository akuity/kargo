import generateCalendar from 'antd/es/calendar/generateCalendar';
import dateFnsGenerateConfig from 'rc-picker/es/generate/dateFns';

export const Calendar = generateCalendar<Date>(dateFnsGenerateConfig);
