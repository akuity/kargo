import { ProjectsList } from '@features/project/list/projects-list';
import { ButtonIcon, PageTitle } from '@features/ui';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { Button } from 'antd';

export const Projects = () => (
  <>
    <PageTitle title='Projects'>
      <Button type='primary' icon={<ButtonIcon icon={faPlus} />}>
        Add Project
      </Button>
    </PageTitle>
    <ProjectsList />
  </>
);
