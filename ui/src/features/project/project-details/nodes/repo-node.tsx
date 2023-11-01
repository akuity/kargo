import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faBuilding } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import { urlWithProtocol } from '@ui/utils/url';

import { NodeType, NodesRepoType } from '../types';

import * as styles from './repo-node.module.less';

const MAX_CHARS = 19;

type Props = {
  nodeData: NodesRepoType;
  height: number;
};

const name = {
  [NodeType.REPO_IMAGE]: 'Image',
  [NodeType.REPO_GIT]: 'Git',
  [NodeType.REPO_CHART]: 'Chart',
  [NodeType.WAREHOUSE]: 'Warehouse'
};

const ico = {
  [NodeType.REPO_IMAGE]: faDocker,
  [NodeType.REPO_GIT]: faGit,
  [NodeType.WAREHOUSE]: faBuilding
};

export const RepoNode = ({ nodeData, height }: Props) => (
  <div style={{ height }} className={styles.node}>
    <h3 className='flex justify-between'>
      <span>{name[nodeData.type]}</span>
      {nodeData.type !== NodeType.REPO_CHART && <FontAwesomeIcon icon={ico[nodeData.type]} />}
    </h3>
    <div className={styles.body}>
      {(nodeData.type === NodeType.REPO_IMAGE || nodeData.type === NodeType.REPO_GIT) && (
        <RepoNodeBody label='Repo URL' value={nodeData.data.repoUrl} />
      )}
      {nodeData.type === NodeType.REPO_CHART && (
        <RepoNodeBody label='Registry URL' value={nodeData.data.registryUrl} />
      )}
      {nodeData.type === NodeType.WAREHOUSE && (
        <div className='text-center font-semibold mt-4'>{nodeData.data as string}</div>
      )}
    </div>
  </div>
);

const RepoNodeBody = ({ label, value }: { label: string; value: string }) => (
  <>
    <div>{label}</div>
    <Tooltip title={value}>
      <a href={urlWithProtocol(value)} className={styles.value} target='_blank' rel='noreferrer'>
        {value.length > MAX_CHARS && '...'}
        {value.substring(value.length - MAX_CHARS)}
      </a>
    </Tooltip>
  </>
);
