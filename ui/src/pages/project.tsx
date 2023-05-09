import { Drawer } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import environmentsData from '../../demo/environments.json';

import { EnvironmentPage } from './environment';

export const Project = () => {
  const { name, environmentName: initEnvironmentName } = useParams();
  const environments: Environment[] = environmentsData?.items || [];
  const environmentsByName = environments.reduce((acc, environment) => {
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
      <Drawer open={currentEnvironment !== null} onClose={() => closeEnvironment()} width={'80%'}>
        <EnvironmentPage environment={environmentsByName[currentEnvironment || '']} />
      </Drawer>
      <h1>{name}</h1>
      <h2>Environments</h2>
      {environments.map((environment) => (
        <EnvironmentItem
          key={environment.metadata.name}
          environment={environment}
          onClick={() => openEnvironment(environment?.metadata.name)}
        />
      ))}
    </div>
  );
};

export interface Environment {
  metadata: any;
  status: any;
  spec: {
    subscriptions: any[];
  } & any;
}

const EnvironmentItem = (props: { environment: Environment; onClick: () => void }) => {
  const { environment } = props;
  return (
    <div key={environment.metadata.name} onClick={props.onClick}>
      {environment.metadata.name}
    </div>
  );
};
