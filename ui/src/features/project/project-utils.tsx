import { faBoxes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Space } from 'antd';
import { generatePath, Link, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';

export const useProjectBreadcrumbs = () => {
  const { name } = useParams();

  return [
    {
      title: (
        <Link to={paths.projects}>
          <Space>
            <FontAwesomeIcon icon={faBoxes} />
            Projects
          </Space>
        </Link>
      )
    },
    {
      title: <Link to={generatePath(paths.project, { name })}>{name}</Link>
    }
  ];
};
