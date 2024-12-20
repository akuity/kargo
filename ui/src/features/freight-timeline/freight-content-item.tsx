import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { TruncateMiddle } from '../common/truncate-middle';

export const FreightContentItem = (props: {
  icon: IconDefinition;
  overlay?: React.ReactNode;
  title?: string;
  href?: string;
  children?: string;
  horizontal?: boolean;
  dark?: boolean;
  highlighted: boolean;
  linkClass: string;
  // don't truncate any content
  fullContentVisibility?: boolean;
}) => {
  const {
    horizontal,
    dark,
    highlighted,
    overlay,
    title,
    icon,
    href,
    children,
    linkClass,
    fullContentVisibility
  } = props;

  const _children = useMemo(() => {
    if (fullContentVisibility) {
      return children;
    }

    return <TruncateMiddle>{children || ''}</TruncateMiddle>;
  }, [fullContentVisibility, children]);

  return (
    <Tooltip
      className={classNames('min-w-0 flex items-center justify-center my-1 rounded', {
        'flex-col p-1 w-full': !horizontal,
        'max-w-60': horizontal && !fullContentVisibility,
        'mr-2 p-2 flex-shrink': horizontal,
        'bg-black text-white': dark,
        'bg-white': !dark && highlighted && !horizontal,
        'border border-solid border-gray-200': !dark && !highlighted && !horizontal,
        'bg-gray-200': !dark && horizontal
      })}
      overlay={overlay}
      title={title}
    >
      <FontAwesomeIcon
        icon={icon}
        style={{ fontSize: '14px' }}
        className={classNames('px-1', {
          'mb-2': !horizontal,
          'mr-2': horizontal
        })}
      />
      {href ? (
        <a target='_blank' className={linkClass}>
          {_children}
        </a>
      ) : (
        _children
      )}
    </Tooltip>
  );
};
