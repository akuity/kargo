import { useQuery } from '@tanstack/react-query';
import { Empty } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { listStages } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { StageItem } from './stage-item';

export const ProjectDetails = () => {
  const { name } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(listStages.useQuery({ project: name }));

  if (isLoading) return <LoadingState />;

  if (!data || data.stages.length === 0) return <Empty />;

  return (
    <>
      {(data?.stages || []).map((stage) => (
        <StageItem
          key={stage.metadata?.name}
          stage={stage}
          onClick={() =>
            stage?.metadata?.name &&
            navigate(generatePath(paths.stage, { name, stageName: stage.metadata.name }))
          }
        />
      ))}
    </>
  );
};
