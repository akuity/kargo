import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import classNames from 'classnames';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';
import { urlForImage } from '@ui/utils/url';

import { CommitInfo } from '../common/commit-info';

import { FreightContentItem } from './freight-content-item';

export const FreightContents = (props: {
  freight?: Freight;
  highlighted: boolean;
  horizontal?: boolean;
  dark?: boolean;
  // don't truncate any content
  fullContentVisibility?: boolean;
}) => {
  const { freight, highlighted, horizontal, dark } = props;
  const linkClass = `${highlighted ? 'text-blue-500' : 'text-gray-400'} hover:text-blue-400 hover:underline max-w-full min-w-0 flex-shrink`;

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
          dark={dark}
          horizontal={horizontal}
          linkClass={linkClass}
          highlighted={highlighted}
          key={c.id}
          overlay={<CommitInfo commit={c} />}
          icon={faGitAlt}
          href={`${c.repoURL?.replace('.git', '')}/commit/${c.id}`}
          fullContentVisibility={props.fullContentVisibility}
        >
          {c.tag && c.tag.length > 12
            ? c.tag.substring(0, 12) + '...'
            : c.tag || c.id?.substring(0, 6)}
        </FreightContentItem>
      ))}
      {(freight?.images || []).map((i) => (
        <FreightContentItem
          dark={dark}
          horizontal={horizontal}
          linkClass={linkClass}
          highlighted={highlighted}
          key={`${i.repoURL}:${i.tag}`}
          title={`${i.repoURL}:${i.tag}`}
          icon={faDocker}
          href={urlForImage(i.repoURL || '')}
          fullContentVisibility={props.fullContentVisibility}
        >
          {`${props.horizontal ? i.repoURL + ':' : ''}${i.tag}`}
        </FreightContentItem>
      ))}
      {(freight?.charts || []).map((c) => (
        <FreightContentItem
          dark={dark}
          horizontal={horizontal}
          linkClass={linkClass}
          highlighted={highlighted}
          key={`${c.repoURL}:${c.version}`}
          title={`${c.repoURL}:${c.version}`}
          fullContentVisibility={props.fullContentVisibility}
          icon={faAnchor}
        >
          {c.version}
        </FreightContentItem>
      ))}
    </div>
  );
};
