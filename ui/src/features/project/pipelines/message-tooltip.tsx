import { SizeProp } from '@fortawesome/fontawesome-svg-core';
import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

export const MessageTooltip = ({
  message,
  copyText,
  icon,
  size = '1x',
  spin,
  iconColor,
  iconClassName
}: {
  message?: React.ReactNode;
  copyText?: string;
  icon: IconDefinition;
  spin?: boolean;
  size?: SizeProp;
  iconColor?: string;
  iconClassName?: string;
}) => {
  return (
    <Tooltip
      title={
        <div className='flex overflow-y-scroll text-wrap max-h-48'>
          <FontAwesomeIcon
            icon={icon}
            className='mr-2 mt-1 pl-1'
            color={iconColor}
            size={size}
            spin={spin}
          />
          <div
            className='cursor-pointer min-w-0'
            onClick={() => {
              if (copyText) {
                navigator.clipboard.writeText(copyText);
              }
            }}
          >
            {message}
          </div>
        </div>
      }
    >
      <FontAwesomeIcon icon={icon} color={iconColor} size={size} className={iconClassName} />
    </Tooltip>
  );
};
