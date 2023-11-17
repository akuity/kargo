import { faPalette, faWandSparkles } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';
import { useParams } from 'react-router-dom';

import { useModal } from '@ui/features/common/modal/use-modal';
import { ProjectDetails } from '@ui/features/project/project-details/project-details';
import { CreateStageModal } from '@ui/features/stage/create-stage-modal';
import { clearColors } from '@ui/features/stage/utils';

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

          <Tooltip title='Reassign Stage Colors'>
            <Button
              type='default'
              className='mr-2'
              onClick={() => {
                clearColors(name || '');
                window.location.reload();
              }}
            >
              <FontAwesomeIcon icon={faPalette} />
            </Button>
          </Tooltip>
          <Button
            type='primary'
            onClick={() => show()}
            icon={<FontAwesomeIcon icon={faWandSparkles} size='1x' />}
          >
            Create
          </Button>
        </div>
      </div>
      <ProjectDetails />
    </div>
  );
};
