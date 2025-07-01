import { faHeart } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Divider, Flex, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import { Project } from '@ui/gen/api/v1alpha1/generated_pb';

import * as styles from './project-item.module.less';

export const ProjectItem = ({ project }: { project?: Project }) => {
  const stagesStats = project?.status?.stats?.stages;
  const warehousesStats = project?.status?.stats?.warehouses;

  return (
    <Link
      className={styles.tile}
      to={generatePath(paths.project, { name: project?.metadata?.name })}
    >
      <Typography.Title level={4} className='!mb-1'>
        {project?.metadata?.name}
      </Typography.Title>
      <Typography.Paragraph type='secondary' className='!mb-0'>
        {project?.metadata?.annotations?.[DESCRIPTION_ANNOTATION_KEY]}
      </Typography.Paragraph>
      <Divider className='my-3' />
      <Flex vertical gap={4}>
        <div>
          <Typography.Text type='secondary' className='inline-block w-28'>
            Stages
          </Typography.Text>
          <Typography.Text
            type={
              (stagesStats?.health?.healthy === stagesStats?.count && stagesStats?.count) || 0 > 0
                ? 'success'
                : 'secondary'
            }
          >
            <FontAwesomeIcon icon={faHeart} />
          </Typography.Text>{' '}
          {stagesStats?.health?.healthy}/{stagesStats?.count}
        </div>
        <div>
          <Typography.Text type='secondary' className='inline-block w-28'>
            Warehouses
          </Typography.Text>
          <Typography.Text
            type={
              (warehousesStats?.health?.healthy === warehousesStats?.count &&
                warehousesStats?.count) ||
              0 > 0
                ? 'success'
                : 'secondary'
            }
          >
            <FontAwesomeIcon icon={faHeart} />
          </Typography.Text>{' '}
          {warehousesStats?.health?.healthy}/{warehousesStats?.count}
        </div>
      </Flex>
    </Link>
  );
};
