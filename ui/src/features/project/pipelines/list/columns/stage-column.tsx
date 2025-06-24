import { faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Input, Space } from 'antd';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  Filter,
  useFilterContext
} from '@ui/features/project/pipelines/list/context/filter-context';
import {
  isStageControlFlow,
  useStageHeaderStyle
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const stageColumn = (filter: Filter): ColumnType<Stage> => ({
  title: 'Stage',
  width: '20%',
  render: (_, stage) => {
    const stageHeader = useStageHeaderStyle(stage);

    const background = stageHeader?.backgroundColor;

    return (
      <Link
        to={generatePath(paths.stage, {
          name: stage?.metadata?.namespace,
          stageName: stage?.metadata?.name
        })}
      >
        {!!background && <Badge color={background} className='mr-2' />}
        {stage?.metadata?.name}
        {isStageControlFlow(stage) ? <span className='text-xs ml-1'>(Control Flow)</span> : ''}
      </Link>
    );
  },
  filterDropdown: (props) => {
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
  },
  onFilter: (value, record) => {
    return !!record?.metadata?.name?.toLowerCase?.()?.includes((value as string).toLowerCase());
  },
  filteredValue: filter?.stage ? [filter.stage] : null,
  filtered: !!filter?.stage,
  sorter: (stage1, stage2) =>
    stage1.metadata?.name?.localeCompare(stage2?.metadata?.name || '') || 0
});
