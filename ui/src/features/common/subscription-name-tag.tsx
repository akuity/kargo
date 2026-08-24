import { Tag } from 'antd';
import classNames from 'classnames';

export const SubscriptionNameTag = ({ name, className }: { name?: string; className?: string }) => {
  if (!name) {
    return null;
  }

  return (
    <Tag
      bordered={false}
      color='blue'
      title={`subscription: ${name}`}
      className={classNames(className)}
    >
      {name}
    </Tag>
  );
};
