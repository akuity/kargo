import { faTag } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag } from 'antd';
import classNames from 'classnames';

// SubscriptionNameTag renders the optional name of a Warehouse subscription. A
// subscription carries its name onto everything it produces -- discovered
// artifacts and Freight -- so the same name surfaces in many unrelated views.
// Rendering it through this one component keeps it recognizable as a
// subscription name wherever it appears. Renders nothing when there is no name,
// so callers can drop it in without guarding.
export const SubscriptionNameTag = ({
  name,
  className
}: {
  name?: string;
  className?: string;
}) => {
  if (!name) {
    return null;
  }

  return (
    <Tag
      bordered={false}
      color='blue'
      title={`subscription: ${name}`}
      className={classNames('font-mono', className)}
    >
      <FontAwesomeIcon icon={faTag} className='mr-1' />
      {name}
    </Tag>
  );
};
