import { PageTitle } from '@ui/features/common';
import { ProjectsList } from '@ui/features/project/list/projects-list';

export const Projects = () => (
  <div className='p-6'>
    <PageTitle title='Projects' />
    <ProjectsList />
  </div>
);
