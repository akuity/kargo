import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import { Freight, GitCommit } from '@ui/gen/v1alpha1/types_pb';

export const FreightContents = (props: {
  freight?: Freight;
  highlighted: boolean;
  promoting: boolean;
}) => {
  const { freight, highlighted, promoting } = props;

  const FreightContentItem = (
    props: {
      icon: IconDefinition;
      overlay?: React.ReactNode;
      title?: string;
    } & React.PropsWithChildren
  ) => (
    <Tooltip
      className={`flex items-center my-1 flex-col bg-neutral-800 rounded p-1 ${
        promoting && highlighted ? 'bg-transparent' : ''
      }`}
      overlay={props.overlay}
      title={props.title}
    >
      <FontAwesomeIcon icon={props.icon} className={`px-1 text-lg mb-2`} />
      {props.children}
    </Tooltip>
  );

  return (
    <div
      className={`hover:text-white flex flex-col justify-center items-center font-mono text-xs flex-shrink min-w-min w-full ${
        highlighted ? 'text-white' : 'text-gray-500'
      }`}
    >
      {(freight?.commits || []).map((c) => (
        <FreightContentItem key={c.id} overlay={<CommitInfo commit={c} />} icon={faGit}>
          <a
            href={`${c.repoUrl.replace('.git', '')}/commit/${c.id}`}
            target='_blank'
            className={`${highlighted ? 'text-blue-200' : 'text-gray-500'} hover:text-blue-300`}
          >
            {c.id.substring(0, 6)}
          </a>
        </FreightContentItem>
      ))}
      {(freight?.images || []).map((i) => (
        <FreightContentItem
          key={`${i.repoUrl}:${i.tag}`}
          title={`${i.repoUrl}:${i.tag}`}
          icon={faDocker}
        >
          <div>{i.tag}</div>
        </FreightContentItem>
      ))}
    </div>
  );
};

const CommitInfo = ({ commit }: { commit: GitCommit }) => (
  <div className='grid grid-cols-2'>
    <div>Repo:</div>
    <div>
      <a href={commit.repoUrl}>{commit.repoUrl}</a>
    </div>
    <div>Branch:</div>
    <div>{commit.branch}</div>
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
