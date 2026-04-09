import { useQuery } from '@connectrpc/connect-query';
import { faStar, faUser } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Pagination, Space, Tag, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { LoadingState } from '@ui/features/common';
import { listProjects } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { useLocalStorage } from '../../../utils/use-local-storage';

import { ProjectItem } from './project-item/project-item';
import { ProjectListFilter } from './project-list-filter';
import * as styles from './projects-list.module.less';
import { useStarProjects } from './use-star-projects';

const PAGE_SIZE_KEY = 'projects-page-size';
const PAGE_NUMBER_KEY = 'projects-page-number';

export const ProjectsList = () => {
  const [pageSize, setPageSize] = useLocalStorage(PAGE_SIZE_KEY, 10);
  const [page, setPage] = useLocalStorage(PAGE_NUMBER_KEY, 1);
  const [filter, setFilter] = useState('');
  const [starredProjectsView, setStarredProjectsView] = useLocalStorage(
    'starred-projects-view',
    false
  );
  const [myProjectsView, setMyProjectsView] = useLocalStorage('my-projects-view', false);

  const [starred, toggleStar] = useStarProjects();

  const { data, isLoading } = useQuery(listProjects, {
    pageSize: pageSize,
    page: page - 1,
    filter,
    uid: starredProjectsView ? starred : [],
    mine: myProjectsView || undefined
  });

  useEffect(() => {
    if (data && page > Math.ceil(data.total / pageSize)) {
      setPage(Math.ceil(data.total / pageSize) || 1);
    }
  }, [data, page, pageSize, setPage]);

  const handlePaginationChange = (newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  };

  const handleFilterChange = (newFilter: string) => {
    setFilter(newFilter);
    setPage(1);
  };

  if (isLoading) return <LoadingState />;

  const isEmpty = !data || data.projects.length === 0;

  if (isEmpty) {
    return (
      <>
        <Flex align='center' className='mb-20' gap={8}>
          <ProjectListFilter onChange={handleFilterChange} init={filter} />
          <Space className='ml-auto'>
            <Tooltip title='Shows projects you have been explicitly granted access to. Broad system-level permissions (e.g. kargo-admin) do not qualify.'>
              <Tag.CheckableTag
                checked={myProjectsView}
                onChange={(checked) => {
                  setMyProjectsView(checked);
                  setPage(1);
                }}
              >
                <FontAwesomeIcon icon={faUser} className='mr-1' />
                My Projects
              </Tag.CheckableTag>
            </Tooltip>
            <Tag.CheckableTag
              checked={starredProjectsView}
              onChange={(checked) => {
                setStarredProjectsView(checked);
                setPage(1);
              }}
            >
              <FontAwesomeIcon icon={faStar} className='mr-1' />
              Starred Projects
            </Tag.CheckableTag>
          </Space>
        </Flex>
        <Empty
          description={
            myProjectsView
              ? 'No projects are directly assigned to your account. Disable this filter to see all projects.'
              : undefined
          }
        />
      </>
    );
  }

  return (
    <>
      <Flex align='center' className='mb-6' gap={8}>
        <ProjectListFilter onChange={handleFilterChange} init={filter} />
        <Space className='ml-auto'>
          <Tooltip title='Shows projects you have been explicitly granted access to. Broad system-level permissions (e.g. kargo-admin) do not qualify.'>
            <Tag.CheckableTag
              checked={myProjectsView}
              onChange={(checked) => {
                setMyProjectsView(checked);
                setPage(1);
              }}
            >
              <FontAwesomeIcon icon={faUser} className='mr-1' />
              My Projects
            </Tag.CheckableTag>
          </Tooltip>
          <Tag.CheckableTag
            checked={starredProjectsView}
            onChange={(checked) => {
              setStarredProjectsView(checked);
              setPage(1);
            }}
          >
            <FontAwesomeIcon icon={faStar} className='mr-1' />
            Starred Projects
          </Tag.CheckableTag>
        </Space>
      </Flex>
      <div className={styles.list}>
        {data.projects.map((proj) => (
          <ProjectItem
            key={proj?.metadata?.name}
            project={proj}
            starred={starred.includes(proj?.metadata?.uid || '')}
            onToggleStar={(id) => toggleStar(id)}
          />
        ))}
      </div>
      <Flex justify='flex-end' className='mt-8'>
        <Pagination
          total={data?.total || 0}
          className='ml-auto flex-shrink-0'
          pageSize={pageSize}
          current={page}
          onChange={handlePaginationChange}
          showSizeChanger
          hideOnSinglePage
        />
      </Flex>
    </>
  );
};
