import { useQuery } from '@connectrpc/connect-query';
import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { AutoComplete, Button } from 'antd';
import { useState } from 'react';

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

  return (
    <div className='flex items-center w-2/3'>
      <AutoComplete
        placeholder='Filter...'
        options={data?.projects.map((p) => ({ value: p.metadata?.name }))}
        onChange={setFilter}
        className='w-full mr-2'
        value={filter}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            onChange(filter);
          }
        }}
      />
      <Button type='primary' onClick={() => onChange(filter)}>
        <FontAwesomeIcon icon={faMagnifyingGlass} />
      </Button>
    </div>
  );
};
