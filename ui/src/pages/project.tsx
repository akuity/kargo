import { Typography } from 'antd';
import React from 'react';
import { useParams } from 'react-router-dom';

import { ProjectDetails } from '@ui/features/project/project-details/project-details';

import { StageDetails } from '../features/stage/stage-details';

export const Project = () => {
  const { name } = useParams();

  return (
    <>
      <Typography.Title level={1}>{name}</Typography.Title>
      <Typography.Title level={3} className='!mt-0 !mb-6'>
        Stages
      </Typography.Title>
      <ProjectDetails />
      <StageDetails />
    </>
  );
};
