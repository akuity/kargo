import { faCode, faPlus, faWandMagicSparkles } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex } from 'antd';
import { useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PageTitle } from '@ui/features/common';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { useModal } from '@ui/features/common/modal/use-modal';
import { CreateProjectModal } from '@ui/features/project/list/create-project-modal';
import { ProjectsList } from '@ui/features/project/list/projects-list';

export const Projects = () => {
  useDocumentTitle(['Projects']);
  const navigate = useNavigate();
  const { show } = useModal((p) => <CreateProjectModal {...p} />);

  return (
    <div className='p-6'>
      <Flex justify='space-between'>
        <PageTitle title='Projects' />
        <Dropdown
          trigger={['click']}
          menu={{
            items: [
              {
                key: 'quick',
                icon: <FontAwesomeIcon icon={faCode} />,
                label: 'Quick create',
                onClick: () => show()
              },
              {
                key: 'guided',
                icon: <FontAwesomeIcon icon={faWandMagicSparkles} />,
                label: 'Guided setup',
                onClick: () => navigate(paths.createProjectGuided)
              }
            ]
          }}
        >
          <Button type='primary' icon={<FontAwesomeIcon icon={faPlus} size='1x' />}>
            New Project
          </Button>
        </Dropdown>
      </Flex>
      <ProjectsList />
    </div>
  );
};
