import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import { Freight, GitCommit } from '@ui/gen/v1alpha1/generated_pb';

export const FreightContents = (props: { freight?: Freight; highlighted: boolean }) => {
  const { freight, highlighted } = props;

  const FreightContentItem = (
    props: {
      icon: IconDefinition;
      overlay?: React.ReactNode;
      title?: string;
    } & React.PropsWithChildren
  ) => (
    <Tooltip
      className={`flex items-center my-1 flex-col bg-neutral-300 rounded p-1 w-full overflow-x-hidden`}
      overlay={props.overlay}
      title={props.title}
    >
      <FontAwesomeIcon icon={props.icon} className={`px-1 text-lg mb-2`} />
      {props.children}
    </Tooltip>
  );

  return (
    <div
      className={`flex flex-col justify-start items-center font-mono text-xs flex-shrink min-w-min w-20 overflow-y-auto max-h-full ${
        highlighted ? 'text-gray-700 hover:text-gray-800' : 'text-gray-400 hover:text-gray-500'
      }`}
    >
      {(freight?.commits || []).map((c) => (
        <FreightContentItem key={c.id} overlay={<CommitInfo commit={c} />} icon={faGit}>
          <a
            href={`${c.repoURL?.replace('.git', '')}/commit/${c.id}`}
            target='_blank'
            className={`${highlighted ? 'text-blue-200' : 'text-gray-500'} hover:text-blue-300`}
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
          <div>{i.tag}</div>
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
