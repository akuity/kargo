import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faBuilding, faCircleNotch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import { urlForImage } from '@ui/utils/url';

import { NodeType, NodesRepoType } from '../types';

import * as styles from './repo-node.module.less';

type Props = {
  nodeData: NodesRepoType;
  children?: React.ReactNode;
};

const ico = {
  [NodeType.REPO_IMAGE]: faDocker,
  [NodeType.REPO_GIT]: faGit,
  [NodeType.WAREHOUSE]: faBuilding,
  [NodeType.REPO_CHART]: faAnchor
};

export const RepoNode = ({ nodeData, children }: Props) => {
  const type = nodeData.type;
  const value =
    type === NodeType.REPO_CHART
      ? nodeData.data.repoUrl
      : type === NodeType.WAREHOUSE
        ? nodeData.data
        : nodeData.data.repoUrl;
  return (
    <div className={styles.node}>
      <h3 className='flex justify-between gap-2'>
        <div className='text-ellipsis whitespace-nowrap overflow-x-hidden pb-1'>
          {nodeData.type === NodeType.WAREHOUSE ? nodeData.data : 'Subscription'}
        </div>
        <div className='flex items-center'>
          {nodeData.refreshing && <FontAwesomeIcon icon={faCircleNotch} spin className='mr-2' />}
          {type && <FontAwesomeIcon icon={ico[type]} />}
        </div>
      </h3>
      <div className={styles.body}>
        {nodeData.type !== NodeType.WAREHOUSE && (
          <div className='mb-2'>
            Repo URL
            <Tooltip title={value}>
              <a
                href={
                  nodeData.type === NodeType.REPO_IMAGE
                    ? urlForImage(value)
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
        )}

        {children}
      </div>
    </div>
  );
};
