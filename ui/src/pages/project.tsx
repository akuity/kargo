import { paths } from '@config/paths';
import { transport } from '@config/transport';
import { HealthStatusIcon } from '@features/ui/health-status-icon/health-status-icon';
import { listStages } from '@gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@gen/v1alpha1/generated_pb';
import { useQuery } from '@tanstack/react-query';
import { Drawer, Typography } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { StageDetails } from '../features/stage/stage-details';

import * as styles from './project.module.less';

export const Project = () => {
  const { name, stageName } = useParams();
  const { data, refetch } = useQuery(listStages.useQuery({ project: name }, { transport }));

  const stagesByName = (data?.stages || []).reduce((acc, stage) => {
    if (stage.metadata?.name) {
      acc[stage.metadata?.name] = stage;
    }
    return acc;
  }, {} as Record<string, Stage>);
  const [currentStage, setCurrentStage] = React.useState<string | null>(stageName || null);

  const navigate = useNavigate();

  const openStage = (stageName: string) => {
    setCurrentStage(stageName);
    navigate(generatePath(paths.stage, { name, stageName }));
  };

  const closeStage = () => {
    setCurrentStage(null);
    navigate(generatePath(paths.project, { name }));
  };

  React.useEffect(() => {
    if (stageName) {
      openStage(stageName);
    }
  }, [stageName]);

  return (
    <div>
      <Drawer
        open={currentStage !== null}
        onClose={() => closeStage()}
        width={'80%'}
        closable={false}
      >
        <StageDetails stage={stagesByName[currentStage || '']} refetch={refetch} />
      </Drawer>
      <Typography.Title level={1}>{name}</Typography.Title>
      <Typography.Title level={3} className='!mt-0 !mb-6'>
        Stages
      </Typography.Title>
      {(data?.stages || []).map((stage) => (
        <StageItem
          key={stage.metadata?.name}
          stage={stage}
          onClick={() => stage?.metadata?.name && openStage(stage.metadata.name)}
        />
      ))}
    </div>
  );
};

const StageItem = (props: { stage: Stage; onClick: () => void }) => {
  const { stage } = props;
  return (
    <div key={stage.metadata?.name} onClick={props.onClick} className={styles.item}>
      <HealthStatusIcon
        health={stage.status?.currentState?.health}
        style={{ marginRight: '12px' }}
      />
      {stage.metadata?.name}
    </div>
  );
};
