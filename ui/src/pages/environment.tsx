import {
  faExternalLinkAlt,
  faHeart,
  faHeartBroken,
  faQuestionCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { CSSProperties } from 'react';
import { Link, useParams } from 'react-router-dom';

import * as styles from './environment.module.less';
import { Environment } from './project';

export const EnvironmentPage = (props: { environment: Environment }) => {
  const { environment } = props;
  const { name: projectName } = useParams();
  return (
    <div>
      <div className={styles.header}>
        <div className={styles.envTitleContainer}>
          <div className={styles.envTitle}>
            <HealthStatusIcon
              health={environment?.status.currentState.health}
              style={{ marginRight: '10px', marginTop: '5px' }}
            />
            <div>{environment?.metadata.name}</div>
          </div>
          <div className={styles.envTitleLabel}>ENVIRONMENT</div>
        </div>
        <div>{projectName}</div>
      </div>

      <Subscriptions
        subscriptions={environment?.spec.subscriptions}
        curProject={projectName || ''}
      />
    </div>
  );
};

interface HealthStatus {
  status: string;
  statusReason: string;
}

const iconForHealthStatus = (status: string): IconDefinition => {
  switch (status) {
    case 'Healthy':
      return faHeart;
    case 'Unhealthy':
      return faHeartBroken;
    case 'Unknown':
      return faQuestionCircle;
    default:
      return faQuestionCircle;
  }
};

const colorForHealthStatus = (status: string): string => {
  switch (status) {
    case 'Healthy':
      return '#52c41a';
    case 'Unhealthy':
      return '#f5222d';
    case 'Unknown':
      return '#faad14';
    default:
      return '#faad14';
  }
};

const HealthStatusIcon = (props: { health: HealthStatus; style?: CSSProperties }) => {
  return (
    props.health && (
      <Tooltip title={props.health.statusReason}>
        <FontAwesomeIcon
          icon={iconForHealthStatus(props.health.status)}
          style={{
            color: colorForHealthStatus(props.health.status),
            fontSize: '18px',
            ...props.style
          }}
        />
      </Tooltip>
    )
  );
};

interface Subscriptions {
  upstreamEnvs?: any[];
  repos?: {
    git: any[];
    images: any[];
  };
}

const Subscriptions = (props: { curProject: string; subscriptions: Subscriptions }) => {
  const { subscriptions, curProject } = props;
  return (
    subscriptions && (
      <div>
        <div className={styles.sectionTitle}>SUBSCRIPTIONS</div>
        {subscriptions.upstreamEnvs && (
          <div className={styles.subscriptionSection}>
            <div className={styles.subscriptionSectionTitle}>Upstream Environments</div>
            {subscriptions?.upstreamEnvs.map((env: any) => (
              <div key={env.name}>
                <Link to={`/project/${curProject}/environment/${env.name}`} target='_blank'>
                  {env.name}
                  <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                </Link>
                <div>Namespace: {env.namespace}</div>
              </div>
            ))}
          </div>
        )}

        {subscriptions.repos && subscriptions.repos?.git && (
          <div className={styles.subscriptionSection}>
            <div className={styles.subscriptionSectionTitle}>Git Repositories</div>
            {subscriptions?.repos.git.map((gitRepo: any) => (
              <div key={gitRepo.repoURL}>
                URL:
                <a href={gitRepo.repoURL} target='_blank'>
                  {gitRepo.repoURL}
                  <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                </a>
              </div>
            ))}
          </div>
        )}

        {subscriptions.repos && subscriptions.repos?.images && (
          <div className={styles.subscriptionSection}>
            <div className={styles.subscriptionSectionTitle}>Images</div>
            {subscriptions?.repos.images.map((image: any) => (
              <div key={image.repoURL}>
                <div key={image.repoURL}>
                  URL:
                  <a href={image.repoURL} target='_blank'>
                    {image.repoURL}
                    <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                  </a>
                </div>
                <div>Semver Constraint: {image.semverConstraint}</div>
                <div>Update Strategy: {image.updateStrategy}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  );
};
