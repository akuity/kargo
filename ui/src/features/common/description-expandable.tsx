import { faFileLines } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { ExpandableConfig } from 'antd/es/table/interface';

import { HasDescriptionAnnotation, getDescription } from './utils';

export function descriptionExpandable<T extends HasDescriptionAnnotation>(): ExpandableConfig<T> {
  return {
    defaultExpandAllRows: true,
    rowExpandable: (record: T) => {
      return !!getDescription(record);
    },
    expandedRowRender: (record: T) => (
      <div className='font-light text-xs text-gray-500 flex items-center'>
        <FontAwesomeIcon icon={faFileLines} className='mr-3' />
        {getDescription(record)}
      </div>
    )
  };
}
