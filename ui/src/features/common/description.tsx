import { faFileLines } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Spin } from 'antd';
import classNames from 'classnames';

import { HasDescriptionAnnotation, getDescription } from './utils';

export function Description<T extends HasDescriptionAnnotation>({
  item,
  loading,
  className
}: {
  item: T;
  loading: boolean;
  className?: string;
}) {
  const description = getDescription(item);
  return (
    <Spin spinning={loading}>
      {description && (
        <div className={classNames('text-sm text-gray-500 flex items-center', className)}>
          <FontAwesomeIcon icon={faFileLines} className='mr-2' />
          {description}
        </div>
      )}
    </Spin>
  );
}
