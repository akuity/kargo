import { generatePath, Link, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';

// No icon on the Projects crumb: the sidebar rail already shows it, with
// Projects marked as the current section.
export const useProjectBreadcrumbs = () => {
  const { name } = useParams();

  return [
    { title: <Link to={paths.projects}>Projects</Link> },
    { title: <Link to={generatePath(paths.project, { name })}>{name}</Link> }
  ];
};
