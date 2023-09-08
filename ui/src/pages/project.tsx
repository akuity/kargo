import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { Button, Typography } from 'antd';
import { useParams } from 'react-router-dom';

import { ButtonIcon } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
import { ProjectDetails } from '@ui/features/project/project-details/project-details';
import { CreateStageModal } from '@ui/features/stage/create-stage-modal';

export const Project = () => {
  const { name } = useParams();
  const { show } = useModal(name ? (p) => <CreateStageModal {...p} project={name} /> : undefined);

  return (
    <>
      <Typography.Title level={1}>{name}</Typography.Title>
      <div className='flex items-center justify-between mb-6'>
        <Typography.Title level={3} className='!mt-0 !mb-6'>
          Stages
        </Typography.Title>
        <Button type='primary' onClick={() => show()} icon={<ButtonIcon icon={faPlus} size='1x' />}>
          Create
        </Button>
      </div>
      <ProjectDetails />
    </>
  );
};
