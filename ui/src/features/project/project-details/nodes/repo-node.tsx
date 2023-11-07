import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faBuilding } from '@fortawesome/free-solid-svg-icons';
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
  [NodeType.WAREHOUSE]: faBuilding,
  [NodeType.REPO_CHART]: faAnchor
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
        <div className='text-center font-semibold mt-2'>
          {(nodeData?.data?.spec?.subscriptions || []).map((sub, i) => {
            return (
              <div key={`${nodeData.data.metadata?.name}-${i}`}>
                {sub.chart && (
                  <RepoNodeBody
                    label='Registry URL'
                    value={sub.chart.registryUrl}
                    type={NodeType.REPO_CHART}
                  />
                )}
                {sub.image && (
                  <RepoNodeBody
                    label='Repo URL'
                    value={sub.image.repoUrl}
                    type={NodeType.REPO_IMAGE}
                  />
                )}
                {sub.git && (
                  <RepoNodeBody label='Repo URL' value={sub.git.repoUrl} type={NodeType.REPO_GIT} />
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  </div>
);

const RepoNodeBody = ({
  label,
  value,
  type
}: {
  label: string;
  value: string;
  type?: NodeType;
}) => (
  <div className='mb-2'>
    <div>
      {type && <FontAwesomeIcon icon={ico[type]} className='mr-2' />}
      {label}
    </div>
    <Tooltip title={value}>
      <a href={urlWithProtocol(value)} className={styles.value} target='_blank' rel='noreferrer'>
        {value.length > MAX_CHARS && '...'}
        {value.substring(value.length - MAX_CHARS)}
      </a>
    </Tooltip>
  </div>
);
