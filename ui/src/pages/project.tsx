import { faWandSparkles } from '@fortawesome/free-solid-svg-icons';
import { Button } from 'antd';
import { useParams } from 'react-router-dom';

import { ButtonIcon } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
import { ProjectDetails } from '@ui/features/project/project-details/project-details';
import { CreateStageModal } from '@ui/features/stage/create-stage-modal';

export const Project = () => {
  const { name } = useParams();
  const { show } = useModal(name ? (p) => <CreateStageModal {...p} project={name} /> : undefined);

  return (
    <div className='h-full flex flex-col'>
      <div className='p-6'>
        <div className='flex items-center'>
          <div className='mr-auto'>
            <div className='font-semibold mb-1 text-xs text-gray-600'>PROJECT</div>
            <div className='text-2xl font-semibold'>{name}</div>
          </div>

          <Button
            type='primary'
            onClick={() => show()}
            icon={<ButtonIcon icon={faWandSparkles} size='1x' />}
          >
            Create
          </Button>
        </div>
      </div>
      <ProjectDetails />
    </div>
  );
};
