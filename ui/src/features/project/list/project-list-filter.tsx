import { useQuery } from '@connectrpc/connect-query';
import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { AutoComplete, Button } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { listProjects } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

export const ProjectListFilter = ({
  onChange,
  init
}: {
  onChange: (filter: string) => void;
  init?: string;
}) => {
  const { data } = useQuery(listProjects);
  const [filter, setFilter] = useState(init || '');
  const navigate = useNavigate();

  const filteredProjects = data?.projects.filter((p) =>
    p.metadata?.name.toLowerCase().includes(filter.toLowerCase())
  );

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && filteredProjects?.length === 1) {
      const selectedProject = filteredProjects[0].metadata?.name;
      navigate(paths.project.replace(':name', selectedProject));
    }
  };

  const handleSelect = (value: string) => {
    navigate(paths.project.replace(':name', value));
  };

  return (
    <div className='flex items-center w-2/3'>
      <AutoComplete
        placeholder='Filter...'
        options={filteredProjects?.map((p) => ({ value: p.metadata?.name }))}
        onChange={setFilter}
        className='w-full mr-2'
        value={filter}
        onKeyDown={handleKeyDown}
        onSelect={handleSelect}
      />
      <Button type='primary' onClick={() => onChange(filter)}>
        <FontAwesomeIcon icon={faMagnifyingGlass} />
      </Button>
    </div>
  );
};
