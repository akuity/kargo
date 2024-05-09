import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';

import { Freight, GitCommit } from '@ui/gen/v1alpha1/generated_pb';
import { urlForImage } from '@ui/utils/url';

export const FreightContents = (props: {
  freight?: Freight;
  highlighted: boolean;
  horizontal?: boolean;
}) => {
  const { freight, highlighted, horizontal } = props;

  const FreightContentItem = (
    props: {
      icon: IconDefinition;
      overlay?: React.ReactNode;
      title?: string;
    } & React.PropsWithChildren
  ) => (
    <Tooltip
      className={classNames(
        'min-w-0 flex items-center justify-center my-1 bg-neutral-300 rounded',
        {
          'flex-col p-1 w-full': !horizontal,
          'mr-2 p-2 max-w-60 flex-shrink': horizontal
        }
      )}
      overlay={props.overlay}
      title={props.title}
    >
      <FontAwesomeIcon
        icon={props.icon}
        className={classNames('px-1 text-lg', {
          'mb-2': !horizontal,
          'mr-2': horizontal
        })}
      />
      {props.children}
    </Tooltip>
  );

  const linkClass = `${highlighted ? 'text-blue-600' : 'text-gray-400'} hover:text-blue-500 underline hover:underline max-w-full truncate`;

  return (
    <div
      className={classNames(
        'flex justify-start items-center font-mono text-xs flex-shrink max-h-full max-w-full',
        {
          'text-gray-700 hover:text-gray-800': highlighted,
          'text-gray-400 hover:text-gray-500': !highlighted,
          'flex-col w-20 overflow-y-auto': !horizontal
        }
      )}
    >
      {(freight?.commits || []).map((c) => (
        <FreightContentItem key={c.id} overlay={<CommitInfo commit={c} />} icon={faGit}>
          <a
            href={`${c.repoURL?.replace('.git', '')}/commit/${c.id}`}
            target='_blank'
            className={linkClass}
          >
            {c.tag && c.tag.length > 12
              ? c.tag.substring(0, 12) + '...'
              : c.tag || c.id?.substring(0, 6)}
          </a>
        </FreightContentItem>
      ))}
      {(freight?.images || []).map((i) => (
        <FreightContentItem
          key={`${i.repoURL}:${i.tag}`}
          title={`${i.repoURL}:${i.tag}`}
          icon={faDocker}
        >
          <a href={urlForImage(i.repoURL || '')} target='_blank' className={linkClass}>
            {props.horizontal && i.repoURL + ':'}
            {i.tag}
          </a>
        </FreightContentItem>
      ))}
      {(freight?.charts || []).map((c) => (
        <FreightContentItem
          key={`${c.repoURL}:${c.version}`}
          title={`${c.repoURL}:${c.version}`}
          icon={faAnchor}
        >
          <div>{c.version}</div>
        </FreightContentItem>
      ))}
    </div>
  );
};

const CommitInfo = ({ commit }: { commit: GitCommit }) => (
  <div className='grid grid-cols-2'>
    <div>Repo:</div>
    <div>
      <a href={commit.repoURL}>{commit.repoURL}</a>
    </div>
    {commit.branch ? (
      <>
        <div>Branch:</div>
        <div>{commit.branch}</div>
      </>
    ) : commit.tag ? (
      <>
        <div>Tag:</div>
        <div>{commit.tag}</div>
      </>
    ) : null}
    {commit.author && (
      <>
        <div>Author:</div>
        <div>{commit.author}</div>
      </>
    )}
    {commit.message && (
      <>
        <div>Message:</div>
        <div>{commit.message}</div>
      </>
    )}
  </div>
);
