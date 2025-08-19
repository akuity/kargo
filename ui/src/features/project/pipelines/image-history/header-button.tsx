import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';
import classNames from 'classnames';
import { memo } from 'react';

export const HeaderButton = memo(
  ({
    onClick,
    icon,
    selected,
    title
  }: {
    onClick: () => void;
    icon: IconDefinition;
    selected?: boolean;
    title: string;
  }) => (
    <Tooltip title={title} placement='left'>
      <Button
        onClick={onClick}
        className={classNames(
          'p-2 w-7 h-7 flex items-center justify-center rounded-md hover:bg-gray-200 transition-colors',
          selected ? 'bg-blue-100 text-blue-600' : 'text-gray-500 hover:text-gray-700'
        )}
      >
        <FontAwesomeIcon icon={icon} />
      </Button>
    </Tooltip>
  )
);
