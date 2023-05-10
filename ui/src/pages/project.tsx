import { Environment, GetEnvironments } from '@client/mock';
import { HealthStatusIcon } from '@features/ui/health-status-icon/health-status-icon';
import { useQuery } from '@tanstack/react-query';
import { Drawer } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { EnvironmentPage } from './environment';
import * as styles from './project.module.less';

export const Project = () => {
  const { name, environmentName: initEnvironmentName } = useParams();

  const { data: environments } = useQuery<Environment[]>(
    ['environments'],
    async () => await GetEnvironments()
  );
  const environmentsByName = (environments || []).reduce((acc, environment) => {
    acc[environment.metadata.name] = environment;
    return acc;
  }, {} as Record<string, Environment>);
  const [currentEnvironment, setCurrentEnvironment] = React.useState<string | null>(
    initEnvironmentName || null
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

  return (
    <div>
      <Drawer
        open={currentEnvironment !== null}
        onClose={() => closeEnvironment()}
        width={'80%'}
        closable={false}
      >
        <EnvironmentPage environment={environmentsByName[currentEnvironment || '']} />
      </Drawer>
      <h1 className={styles.header}>{name}</h1>
      <h2 className={styles.subHeader}>Environments</h2>
      {(environments || []).map((environment) => (
        <EnvironmentItem
          key={environment.metadata.name}
          environment={environment}
          onClick={() => openEnvironment(environment?.metadata.name)}
        />
      ))}
    </div>
  );
};

const EnvironmentItem = (props: { environment: Environment; onClick: () => void }) => {
  const { environment } = props;
  return (
    <div
      key={environment.metadata.name}
      onClick={props.onClick}
      className={styles.environmentItem}
      style={{ display: 'flex', alignItems: 'center' }}
    >
      <HealthStatusIcon
        health={environment.status?.currentState?.health}
        style={{ marginRight: '8px' }}
      />
      {environment.metadata.name}
    </div>
  );
};
