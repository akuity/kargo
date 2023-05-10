import { Environment, GetPromotionPoliciesForEnvironment, PromotionPolicy } from '@client/mock';
import { HealthStatusIcon } from '@features/ui/health-status-icon/health-status-icon';
import { faExternalLinkAlt, faToggleOff, faToggleOn } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQuery } from '@tanstack/react-query';
import { Typography } from 'antd';
import { Link, useParams } from 'react-router-dom';

import * as styles from './environment.module.less';

export const EnvironmentPage = (props: { environment: Environment }) => {
  const { environment } = props;
  const { name: projectName } = useParams();

  const { data: promotionPolicies } = useQuery<PromotionPolicy[]>(
    ['promotionPolicies', environment],
    async () => (await GetPromotionPoliciesForEnvironment(environment?.metadata.name)) || []
  );

  return (
    <div>
      <div className={styles.header}>
        <div className={styles.envTitleContainer}>
          <div className={styles.envTitle}>
            <HealthStatusIcon
              health={environment?.status?.currentState?.health}
              style={{ marginRight: '10px', marginTop: '5px' }}
            />
            <div>{environment?.metadata.name}</div>
          </div>
          <div className={styles.titleLabel}>ENVIRONMENT</div>
        </div>
        <div className={styles.projectTitleContainer}>
          <div>{projectName}</div>
          <div className={styles.titleLabel}>PROJECT</div>
        </div>
      </div>

      <div className={styles.section}>
        <Subscriptions
          subscriptions={environment?.spec.subscriptions}
          curProject={projectName || ''}
        />
      </div>

      <div className={styles.section}>
        <AvailableStates environment={environment} />
      </div>

      <div className={styles.section}>
        <AssociatedPolicies policies={promotionPolicies || []} />
      </div>
    </div>
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
        <div className={styles.sectionTitle}>Subscriptions</div>
        {subscriptions.upstreamEnvs && (
          <div className={styles.subscriptionSection}>
            <div className={styles.subtitle}>Upstream Environments</div>
            {subscriptions?.upstreamEnvs.map((env: any) => (
              <div key={env.name}>
                <Link
                  to={`/project/${curProject}/environment/${env.name}`}
                  target='_blank'
                  style={{ fontSize: '16px' }}
                >
                  {env.name}
                  <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                </Link>
                <div>
                  <div className={styles.dataLabel}>Namespace</div>
                  {env.namespace}
                </div>
              </div>
            ))}
          </div>
        )}

        {subscriptions.repos && subscriptions.repos?.git && (
          <div className={styles.subscriptionSection}>
            <div className={styles.subtitle}>Git Repositories</div>
            {subscriptions?.repos.git.map((gitRepo: any) => (
              <div key={gitRepo.repoURL}>
                <div className={styles.dataLabel}>URL</div>
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
            <div className={styles.subtitle}>Images</div>
            {subscriptions?.repos.images.map((image: any) => (
              <div key={image.repoURL}>
                <div key={image.repoURL}>
                  <div className={styles.dataLabel}>URL</div>
                  <a href={image.repoURL} target='_blank'>
                    {image.repoURL}
                    <FontAwesomeIcon icon={faExternalLinkAlt} style={{ marginLeft: '5px' }} />
                  </a>
                </div>
                <div>
                  <div className={styles.dataLabel}>Semver Constraint</div>
                  {image.semverConstraint}
                </div>
                <div>
                  <div className={styles.dataLabel}>Update Strategy</div> {image.updateStrategy}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  );
};

const AvailableStates = (props: { environment: Environment }) => {
  const { environment } = props;
  return (
    <div style={{ marginBottom: '1em' }}>
      <div className={styles.sectionTitle}>Available States</div>
      <div>
        {(environment?.status?.availableStates || []).map((state: any) => (
          <div key={state.id}>{JSON.stringify(state)}</div>
        ))}
      </div>
    </div>
  );
};

const AssociatedPolicies = (props: { policies: PromotionPolicy[] }) => {
  const { policies } = props;
  return (
    <div>
      <div className={styles.sectionTitle}>Associated Promotion Policies</div>
      {(policies || []).map((policy: PromotionPolicy) => (
        <div key={policy.metadata.uid} className={styles.promotionPolicy}>
          <div className={styles.subtitle}>{policy.metadata.name}</div>
          <div style={{ marginBottom: '1em' }}>
            <div className={styles.dataLabel}>Auto Promotion</div>
            {policy.enableAutoPromotion ? 'Enabled' : 'Disabled'}
            <FontAwesomeIcon
              icon={policy.enableAutoPromotion ? faToggleOn : faToggleOff}
              style={{ marginLeft: '7px' }}
            />
          </div>
          <div>
            <div style={{ fontSize: '14px', fontWeight: 500, marginBottom: '0.5em' }}>
              Authorized Promoters
            </div>
            {(policy?.authorizedPromoters || []).map((promoter: any) => (
              <div
                key={promoter.name}
                style={{ backgroundColor: 'white', padding: '1em', borderRadius: '5px' }}
              >
                <div>
                  <div className={styles.dataLabel}>Name</div>
                  <Typography.Text code>{promoter.name}</Typography.Text>
                </div>
                <div>
                  <div className={styles.dataLabel}>Subject Type</div>
                  <Typography.Text code>{promoter.subjectType}</Typography.Text>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};
