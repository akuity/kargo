import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex } from 'antd';

import { PageTitle } from '@ui/features/common';
import { useModal } from '@ui/features/common/modal/use-modal';
import { CreateProjectModal } from '@ui/features/project/list/create-project-modal';
import { ProjectsList } from '@ui/features/project/list/projects-list';

export const Projects = () => {
  const { show } = useModal((p) => <CreateProjectModal {...p} />);

  return (
    <div className='p-6'>
      <Flex justify='space-between'>
        <PageTitle title='Projects' />
        <Button
          type='primary'
          onClick={() => show()}
          icon={<FontAwesomeIcon icon={faPlus} size='1x' />}
        >
          New Project
        </Button>
      </Flex>
      <ProjectsList />
    </div>
  );
};
