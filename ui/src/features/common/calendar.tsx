import dateFnsGenerateConfig from '@rc-component/picker/es/generate/dateFns';
import generateCalendar from 'antd/es/calendar/generateCalendar';

export const Calendar = generateCalendar<Date>(dateFnsGenerateConfig);
