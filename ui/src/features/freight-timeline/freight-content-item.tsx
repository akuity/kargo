import { faCode, faHammer, IconDefinition } from '@fortawesome/free-solid-svg-icons';
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
  // source of image
  artifactSource?: string;
  // build date of image
  artifactBuildDate?: string;
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
      <div className='flex gap-2 items-center'>
        <FontAwesomeIcon icon={icon} style={{ fontSize: '14px' }} className={classNames('px-1')} />
        {props.artifactSource && (
          <a
            href={props.artifactSource}
            className={classNames('text-blue-500', {
              'mr-2': horizontal
            })}
            style={{ fontSize: '10px' }}
            onClick={(e) => {
              e.stopPropagation();
            }}
            target='_blank'
          >
            {horizontal && <u>image source</u>}
            <FontAwesomeIcon
              icon={faCode}
              style={{ fontSize: '10px' }}
              className={horizontal ? 'ml-1' : ''}
            />
          </a>
        )}
      </div>
      <div
        className={classNames(
          { 'mt-2 flex-col': !horizontal, 'gap-2': horizontal },
          'flex items-center'
        )}
      >
        {href ? (
          <a target='_blank' className={linkClass}>
            {_children}
          </a>
        ) : (
          _children
        )}
        {!!props.artifactBuildDate && (
          <span className='text-[8px]'>
            <FontAwesomeIcon icon={faHammer} />
            {props.artifactBuildDate}
          </span>
        )}
      </div>
    </Tooltip>
  );
};
