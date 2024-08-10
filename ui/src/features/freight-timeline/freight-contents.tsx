import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';
import { urlForImage } from '@ui/utils/url';

import { CommitInfo } from '../common/commit-info';
import { TruncateMiddle } from '../common/truncate-middle';

export const FreightContents = (props: {
  freight?: Freight;
  highlighted: boolean;
  horizontal?: boolean;
  dark?: boolean;
}) => {
  const { freight, highlighted, horizontal, dark } = props;
  const linkClass = `${highlighted ? 'text-blue-500' : 'text-gray-400'} hover:text-blue-400 hover:underline max-w-full min-w-0 flex-shrink`;

  const FreightContentItem = (props: {
    icon: IconDefinition;
    overlay?: React.ReactNode;
    title?: string;
    href?: string;
    children?: string;
  }) => (
    <Tooltip
      className={classNames('min-w-0 flex items-center justify-center my-1 rounded', {
        'flex-col p-1 w-full': !horizontal,
        'mr-2 p-2 max-w-60 flex-shrink': horizontal,
        'bg-black text-white': dark,
        'bg-white': !dark && highlighted && !horizontal,
        'border border-solid border-gray-200': !dark && !highlighted && !horizontal,
        'bg-gray-200': !dark && horizontal
      })}
      overlay={props.overlay}
      title={props.title}
    >
      <FontAwesomeIcon
        icon={props.icon}
        style={{ fontSize: '14px' }}
        className={classNames('px-1', {
          'mb-2': !horizontal,
          'mr-2': horizontal
        })}
      />
      {props.href ? (
        <a target='_blank' className={linkClass}>
          <TruncateMiddle>{props.children || ''}</TruncateMiddle>
        </a>
      ) : (
        <TruncateMiddle>{props.children || ''}</TruncateMiddle>
      )}
    </Tooltip>
  );

  return (
    <div
      className={classNames(
        'flex justify-start items-center font-mono text-xs flex-shrink max-h-full max-w-full flex-wrap',
        {
          'text-gray-700 hover:text-gray-800': highlighted,
          'text-gray-400 hover:text-gray-500': !highlighted,
          'flex-col w-20 overflow-y-auto flex-grow-0 flex-nowrap': !horizontal
        }
      )}
    >
      {(freight?.commits || []).map((c) => (
        <FreightContentItem
          key={c.id}
          overlay={<CommitInfo commit={c} />}
          icon={faGitAlt}
          href={`${c.repoURL?.replace('.git', '')}/commit/${c.id}`}
        >
          {c.tag && c.tag.length > 12
            ? c.tag.substring(0, 12) + '...'
            : c.tag || c.id?.substring(0, 6)}
        </FreightContentItem>
      ))}
      {(freight?.images || []).map((i) => (
        <FreightContentItem
          key={`${i.repoURL}:${i.tag}`}
          title={`${i.repoURL}:${i.tag}`}
          icon={faDocker}
          href={urlForImage(i.repoURL || '')}
        >
          {`${props.horizontal ? i.repoURL + ':' : ''}${i.tag}`}
        </FreightContentItem>
      ))}
      {(freight?.charts || []).map((c) => (
        <FreightContentItem
          key={`${c.repoURL}:${c.version}`}
          title={`${c.repoURL}:${c.version}`}
          icon={faAnchor}
        >
          {c.version}
        </FreightContentItem>
      ))}
    </div>
  );
};
