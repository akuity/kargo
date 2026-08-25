import { Typography } from 'antd';
import classNames from 'classnames';

export const SubscriptionName = ({ name, className }: { name?: string; className?: string }) => {
  if (!name) {
    return null;
  }

  return (
    <Typography.Text
      type='secondary'
      title={`subscription: ${name}`}
      className={classNames(className)}
    >
      {name}
    </Typography.Text>
  );
};
