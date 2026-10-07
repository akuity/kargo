import dateFnsGenerateConfig from '@rc-component/picker/es/generate/dateFns';
import generatePicker from 'antd/es/date-picker/generatePicker';

export const DatePicker = generatePicker<Date>(dateFnsGenerateConfig);

export const TimePicker = DatePicker.TimePicker;
