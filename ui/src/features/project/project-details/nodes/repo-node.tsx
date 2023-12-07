import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faBuilding } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import { urlWithProtocol } from '@ui/utils/url';

import { NodeType, NodesRepoType } from '../types';

import * as styles from './repo-node.module.less';

type Props = {
  nodeData: NodesRepoType;
  children?: React.ReactNode;
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

export const RepoNode = ({ nodeData, children }: Props) => {
  const type = nodeData.type;
  const value = type === NodeType.REPO_CHART ? nodeData.data.registryUrl : nodeData.data.repoUrl;
  return (
    <div className={styles.node}>
      <h3 className='flex justify-between gap-2'>
        <div className='text-ellipsis whitespace-nowrap overflow-hidden'>
          {nodeData.warehouseName}
        </div>
        {nodeData.type !== NodeType.REPO_CHART && <FontAwesomeIcon icon={faBuilding} />}
      </h3>
      <div className={styles.body}>
        <div className='mb-2'>
          <div className='flex items-center font-semibold text-sm mb-2'>
            {type && <FontAwesomeIcon icon={ico[type]} className='mr-2' />}
            {name[type as NodeType.REPO_CHART | NodeType.REPO_GIT | NodeType.REPO_IMAGE]}
          </div>
          {nodeData.type === NodeType.REPO_CHART ? 'Registry URL' : 'Repo URL'}
          <Tooltip title={value}>
            <a
              href={
                nodeData.type === NodeType.REPO_IMAGE
                  ? urlWithProtocol(value)
                  : `https://${value.replace('https://', '')}`
              }
              className={styles.value}
              target='_blank'
              rel='noreferrer'
            >
              {value}
            </a>
          </Tooltip>
        </div>
        {children}
      </div>
    </div>
  );
};
