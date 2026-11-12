import { Typography } from 'antd';

export const StepTitle = ({ title }: { title?: string }) => (
  <Typography.Text strong ellipsis={{ tooltip: title }} className='max-w-[260px] shrink-0'>
    {title}
  </Typography.Text>
);
