import { faCode, faHammer, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tooltip } from 'antd';
import Link from 'antd/es/typography/Link';
import classNames from 'classnames';
import { useMemo } from 'react';

import { SubscriptionName } from '../common/subscription-name';
import { TruncateMiddle } from '../common/truncate-middle';

export const FreightContentItem = (props: {
  icon?: IconDefinition;
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
  // name of the subscription that discovered the artifact, if it has one
  subscriptionName?: string;
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
    fullContentVisibility,
    subscriptionName
  } = props;

  // The chips are too cramped to always carry the name, so it rides along in
  // the tooltip and only becomes a visible tag where there is room for it.
  const _title = useMemo(() => {
    if (!subscriptionName) {
      return title;
    }

    return title
      ? `${title} (subscription: ${subscriptionName})`
      : `subscription: ${subscriptionName}`;
  }, [title, subscriptionName]);

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
        'bg-white dark:bg-neutral-900': !dark && highlighted && !horizontal,
        'border border-solid border-gray-200 dark:border-neutral-700':
          !dark && !highlighted && !horizontal,
        'bg-gray-200 dark:bg-neutral-700': !dark && horizontal
      })}
      overlay={overlay}
      title={_title}
    >
      <Flex align='center' gap={8}>
        {!!icon && (
          <FontAwesomeIcon
            icon={icon}
            style={{ fontSize: '14px' }}
            className={classNames('px-1')}
          />
        )}
        {props.artifactSource && (
          <Link
            href={props.artifactSource}
            className={classNames({
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
          </Link>
        )}
      </Flex>
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
          <span className='text-[8px] text-center'>
            <FontAwesomeIcon icon={faHammer} />
            {props.artifactBuildDate}
          </span>
        )}
        {fullContentVisibility && (
          <SubscriptionName name={subscriptionName} className='text-[10px]' />
        )}
      </div>
    </Tooltip>
  );
};
