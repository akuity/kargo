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
import { useLocalStorage } from '../../../utils/use-local-storage';

const PAGE_SIZE_KEY = 'projects-page-size';
const PAGE_NUMBER_KEY = 'projects-page-number';
const FILTER_KEY = 'projects-filter';

export const ProjectsList = () => {
    const [pageSize, setPageSize] = useLocalStorage(PAGE_SIZE_KEY, 10);
    const [page, setPage] = useLocalStorage(PAGE_NUMBER_KEY, 1);
    const [filter, setFilter] = useLocalStorage(FILTER_KEY, '');

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
                    showSizeChanger
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
