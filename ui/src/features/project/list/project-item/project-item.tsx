import { IconDefinition, faBullseye, faHeart, faStar } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Divider, Flex, Tooltip, theme, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import type { Project } from '@ui/gen/api/v2/models';

import { type HealthSummary, healthSummary, healthTooltip } from './health-summary';
import * as styles from './project-item.module.less';

export const ProjectItem = ({
  project,
  starred,
  onToggleStar
}: {
  project?: Project;
  starred: boolean;
  onToggleStar: (id: string) => void;
}) => {
  const stagesStats = healthSummary(project?.status?.stats?.stages);
  const warehousesStats = healthSummary(project?.status?.stats?.warehouses);
  const targetsStats = healthSummary(project?.status?.stats?.targets);
  const { token } = theme.useToken();
  const primaryColor = token.colorPrimary;

  return (
    <Link
      className={styles.tile}
      to={generatePath(paths.project, { name: project?.metadata?.name })}
    >
      <Flex align='start' justify='space-between' gap={2}>
        <div>
          <Typography.Title level={4} className='!mb-1'>
            {project?.metadata?.name}
          </Typography.Title>
          <Typography.Paragraph type='secondary' className='!mb-0'>
            {project?.metadata?.annotations?.[DESCRIPTION_ANNOTATION_KEY]}
          </Typography.Paragraph>
        </div>

        <Button
          type='text'
          size='small'
          icon={
            <FontAwesomeIcon
              icon={faStar}
              style={{ color: starred ? primaryColor : '', opacity: starred ? 1 : 0.3 }}
            />
          }
          onClick={(e) => {
            e.preventDefault();
            onToggleStar(project?.metadata?.uid || '');
          }}
        />
      </Flex>
      <Divider className='my-3' />
      <Flex vertical gap={4}>
        <HealthRow kind='Stage' icon={faHeart} summary={stagesStats} />
        <HealthRow kind='Warehouse' icon={faHeart} summary={warehousesStats} />
        {targetsStats.count > 0 && (
          // A Project promoting to a fleet gets one more row, in the same shape
          // as the two above it. The controller leaves the block absent for a
          // Project with no Targets, and a Project whose Stages select no
          // Targets has nothing to count, so neither shows the row.
          <HealthRow kind='Target' icon={faBullseye} summary={targetsStats} />
        )}
      </Flex>
    </Link>
  );
};

const HealthRow = ({
  kind,
  icon,
  summary
}: {
  kind: string;
  icon: IconDefinition;
  summary: HealthSummary;
}) => (
  <div>
    <Typography.Text type='secondary' className='inline-block w-28'>
      {kind}s
    </Typography.Text>
    <Tooltip title={healthTooltip(summary, kind)}>
      <span>
        <Typography.Text type={summary.allHealthy ? 'success' : 'secondary'}>
          <FontAwesomeIcon icon={icon} />
        </Typography.Text>{' '}
        {summary.healthy}/{summary.count}
      </span>
    </Tooltip>
  </div>
);
