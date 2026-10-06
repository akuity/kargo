import generatePicker from 'antd/es/date-picker/generatePicker';
import dateFnsGenerateConfig from 'rc-picker/es/generate/dateFns';

export const DatePicker = generatePicker<Date>(dateFnsGenerateConfig);

export const TimePicker = DatePicker.TimePicker;
