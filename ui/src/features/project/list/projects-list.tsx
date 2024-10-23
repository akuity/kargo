import { createQueryOptions, useQuery, useTransport } from '@connectrpc/connect-query';
import { useQueries } from '@tanstack/react-query';
import { Empty, Pagination } from 'antd';
import { useState, useEffect } from 'react';

import { LoadingState } from '@ui/features/common';
import {
  listProjects,
  listStages
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ProjectItem } from './project-item/project-item';
import { ProjectListFilter } from './project-list-filter';
import * as styles from './projects-list.module.less';

const PAGE_SIZE_KEY = 'projects-page-size';
const PAGE_NUMBER_KEY = 'projects-page-number';
const FILTER_KEY = 'projects-filter';

const getStoredValue = (key: string, defaultValue: number | string) => {
  try {
    const item = localStorage.getItem(key);
    return item ? JSON.parse(item) : defaultValue;
  } catch (error) {
    console.error('Error reading from localStorage:', error);
    return defaultValue;
  }
};

const setStoredValue = (key: string, value: number | string) => {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch (error) {
    console.error('Error writing to localStorage:', error);
  }
};

export const ProjectsList = () => {
  const [pageSize, setPageSize] = useState(() => getStoredValue(PAGE_SIZE_KEY, 9));
  const [page, setPage] = useState(() => getStoredValue(PAGE_NUMBER_KEY, 1));
  const [filter, setFilter] = useState(() => getStoredValue(FILTER_KEY, ''));

  useEffect(() => {
    setStoredValue(PAGE_SIZE_KEY, pageSize);
  }, [pageSize]);

  useEffect(() => {
    setStoredValue(PAGE_NUMBER_KEY, page);
  }, [page]);

  useEffect(() => {
    setStoredValue(FILTER_KEY, filter);
  }, [filter]);

  const { data, isLoading } = useQuery(listProjects, {
    pageSize: pageSize,
    page: page - 1,
    filter
  });

  const transport = useTransport();
  const stageData = useQueries({
    queries: (data?.projects || []).map((proj) => {
      return createQueryOptions(listStages, { project: proj?.metadata?.name }, { transport });
    })
  });

  const handlePaginationChange = (newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  };

  const handleFilterChange = (newFilter: string) => {
    setFilter(newFilter);
    setPage(1);
  };

  if (isLoading) return <LoadingState />;

  if (!data || data.projects.length === 0) return <Empty />;

  return (
    <>
      <div className='flex items-center mb-6'>
        <ProjectListFilter onChange={handleFilterChange} init={filter} />
        <Pagination
          total={data?.total || 0}
          className='ml-auto flex-shrink-0'
          pageSize={pageSize}
          current={page}
          onChange={handlePaginationChange}
        />
      </div>
      <div className={styles.list}>
        {data.projects.map((proj, i) => (
          <ProjectItem
            key={proj?.metadata?.name}
            project={proj}
            stages={stageData[i]?.data?.stages}
          />
        ))}
      </div>
    </>
  );
};
