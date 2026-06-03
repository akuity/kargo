import { faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Input, Space } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';

import { useFilterContext } from '@ui/features/project/pipelines/list/context/filter-context';

export const StageFilterDropdown = (props: FilterDropdownProps) => {
  const filters = useFilterContext();

  const searchTerm = (props.selectedKeys[0] as string) || '';

  const onSearch = (term: string) => filters?.onFilter({ ...filters.filters, stage: term });

  return (
    <div style={{ padding: 8 }}>
      <Input
        placeholder='Search stage'
        value={searchTerm}
        onChange={(e) => props.setSelectedKeys(e.target.value ? [e.target.value] : [])}
        onPressEnter={() => onSearch(searchTerm)}
      />
      <Space>
        <Button
          type='primary'
          size='small'
          className='mt-2'
          icon={<FontAwesomeIcon icon={faSearch} />}
          onClick={() => onSearch(searchTerm)}
        >
          Search
        </Button>
      </Space>
    </div>
  );
};
