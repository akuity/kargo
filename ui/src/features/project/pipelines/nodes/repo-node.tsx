import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faBuilding,
  faCircleNotch,
  faExclamationCircle,
  faExternalLinkAlt
} from '@fortawesome/free-solid-svg-icons';
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
      ? nodeData?.data?.repoURL || ''
      : type === NodeType.WAREHOUSE
        ? nodeData?.data?.metadata?.name || ''
        : nodeData?.data?.repoURL || '';
  return (
    <div className={styles.node}>
      <h3 className='flex justify-between gap-2'>
        <div className='text-ellipsis whitespace-nowrap overflow-x-hidden py-1'>
          {nodeData.type === NodeType.WAREHOUSE ? nodeData.data?.metadata?.name : 'Subscription'}
        </div>
        <div className='flex items-center'>
          {nodeData.refreshing && <FontAwesomeIcon icon={faCircleNotch} spin className='mr-2' />}
          {nodeData.type === NodeType.WAREHOUSE && nodeData?.data?.status?.message && (
            <Tooltip
              title={
                <div className='flex items-center'>
                  <FontAwesomeIcon icon={faExclamationCircle} className='mr-2' />
                  {nodeData?.data?.status?.message}
                </div>
              }
            >
              <FontAwesomeIcon icon={faExclamationCircle} className='mr-1 text-red-600' />
            </Tooltip>
          )}
          {type && <FontAwesomeIcon icon={ico[type]} />}
        </div>
      </h3>
      <div className={styles.body}>
        {nodeData.type !== NodeType.WAREHOUSE && (
          <div className={styles.valueContainer}>
            <div className={styles.repoLabel}>REPO URL</div>
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
                <FontAwesomeIcon icon={faExternalLinkAlt} size='sm' className='mr-2' />
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
