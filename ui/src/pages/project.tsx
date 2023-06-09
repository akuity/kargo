import { transport } from '@config/transport';
import { HealthStatusIcon } from '@features/ui/health-status-icon/health-status-icon';
import { listEnvironments } from '@gen/service/v1alpha1/service-KargoService_connectquery';
import { Environment } from '@gen/v1alpha1/generated_pb';
import { useQuery } from '@tanstack/react-query';
import { Drawer, Typography } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { EnvironmentDetails } from '../features/environment/environment-details';

import * as styles from './project.module.less';

export const Project = () => {
  const { name, environmentName } = useParams();
  const { data } = useQuery(listEnvironments.useQuery({ project: name }, { transport }));

  const environmentsByName = (data?.environments || []).reduce((acc, environment) => {
    if (environment.metadata?.name) {
      acc[environment.metadata?.name] = environment;
    }
    return acc;
  }, {} as Record<string, Environment>);
  const [currentEnvironment, setCurrentEnvironment] = React.useState<string | null>(
    environmentName || null
  );

  const navigate = useNavigate();

  const openEnvironment = (environmentName: string) => {
    setCurrentEnvironment(environmentName);
    navigate(`/project/${name}/environment/${environmentName}`);
  };

  const closeEnvironment = () => {
    setCurrentEnvironment(null);
    navigate(`/project/${name}`);
  };

  React.useEffect(() => {
    if (environmentName) {
      openEnvironment(environmentName);
    }
  }, [environmentName]);

  return (
    <div>
      <Drawer
        open={currentEnvironment !== null}
        onClose={() => closeEnvironment()}
        width={'80%'}
        closable={false}
      >
        <EnvironmentDetails environment={environmentsByName[currentEnvironment || '']} />
      </Drawer>
      <Typography.Title level={1}>{name}</Typography.Title>
      <Typography.Title level={3} className='!mt-0 !mb-6'>
        Environments
      </Typography.Title>
      {(data?.environments || []).map((environment) => (
        <EnvironmentItem
          key={environment.metadata?.name}
          environment={environment}
          onClick={() => environment?.metadata?.name && openEnvironment(environment.metadata.name)}
        />
      ))}
    </div>
  );
};

const EnvironmentItem = (props: { environment: Environment; onClick: () => void }) => {
  const { environment } = props;
  return (
    <div
      key={environment.metadata?.name}
      onClick={props.onClick}
      className={styles.environmentItem}
    >
      <HealthStatusIcon
        health={environment.status?.currentState?.health}
        style={{ marginRight: '12px' }}
      />
      {environment.metadata?.name}
    </div>
  );
};
