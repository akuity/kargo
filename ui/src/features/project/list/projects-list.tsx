import { faStar, faUser } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { keepPreviousData } from '@tanstack/react-query';
import { Divider, Empty, Flex, Pagination, Segmented, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { useEffect, useState } from 'react';

import { useListProjects } from '@ui/gen/api/v2/core/core';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { ProjectItem, ProjectItemSkeleton } from './project-item/project-item';
import { ProjectListFilter } from './project-list-filter';
import { useStarProjects } from './use-star-projects';

const PAGE_SIZE_KEY = 'projects-page-size';
const PAGE_NUMBER_KEY = 'projects-page-number';
const VIEW_KEY = 'projects-view';

// Page sizes that fill whole rows whether the grid is two, three or four cards
// wide.
const PAGE_SIZE_OPTIONS = [12, 24, 48, 96];

// A page at the smallest size, which fills whole rows too.
const SKELETON_COUNT = PAGE_SIZE_OPTIONS[0];

// Columns are at least 360px, but never wider than the list itself: a page too
// narrow for that gets one card that fits rather than one that overflows.
const gridClassName = 'grid grid-cols-[repeat(auto-fill,minmax(min(100%,360px),1fr))] gap-4';

// Which projects the list shows. One at a time: "mine" and "starred" are two
// different questions, and answering both at once rarely helped.
type View = 'all' | 'mine' | 'starred';

const MINE_TOOLTIP =
  'Projects you have been explicitly granted access to. Broad system-level permissions (e.g. kargo-admin) do not qualify.';

export const ProjectsList = () => {
  const [pageSize, setPageSize] = useLocalStorage(PAGE_SIZE_KEY, PAGE_SIZE_OPTIONS[0]);
  const [page, setPage] = useLocalStorage(PAGE_NUMBER_KEY, 1);
  const [filter, setFilter] = useState('');
  const [storedView, setView] = useLocalStorage<View>(VIEW_KEY, 'all');
  // Anything may be in storage, including a view that no longer exists.
  const view: View = storedView === 'mine' || storedView === 'starred' ? storedView : 'all';

  const [starred, toggleStar] = useStarProjects();

  // Nothing starred is nothing to show. Asking anyway would send no UIDs, which
  // the API reads as no filter at all, and list every project.
  const noStars = view === 'starred' && !starred?.length;

  const { data, isLoading, isPlaceholderData } = useListProjects(
    {
      pageSize,
      page: page - 1,
      filter,
      uid: view === 'starred' ? starred : undefined,
      mine: view === 'mine' || undefined
    },
    {
      // The cards stay on screen while a new search or page loads, rather than
      // the list collapsing to placeholders on every keystroke.
      query: { enabled: !noStars, placeholderData: keepPreviousData }
    }
  );

  const projects = noStars ? [] : (data?.data?.items ?? []);
  const total = noStars ? 0 : (data?.data?.total ?? 0);

  // Dimming the previous results says the next ones are on their way, but an
  // empty view has nothing to dim: kept up, it would claim there is nothing in
  // the next view before the answer is in. That case shows the placeholder
  // cards a first visit does. A search that matched nothing stays up, dimmed,
  // while it is refined, rather than flashing placeholders at every pause.
  const loading = isLoading || (isPlaceholderData && !noStars && !filter && projects.length === 0);

  useEffect(() => {
    if (total > 0 && page > Math.ceil(total / pageSize)) {
      setPage(Math.ceil(total / pageSize) || 1);
    }
  }, [total, page, pageSize, setPage]);

  const handleFilterChange = (newFilter: string) => {
    setFilter(newFilter);
    setPage(1);
  };

  const handleViewChange = (newView: View) => {
    setView(newView);
    setPage(1);
  };

  const emptyDescription = () => {
    if (filter) {
      return `No projects match "${filter}"`;
    }
    switch (view) {
      case 'mine':
        return 'No projects are directly assigned to your account. Switch to All to see every project.';
      case 'starred':
        return 'No starred projects yet. Star a project to keep it close at hand.';
      default:
        return 'No projects yet';
    }
  };

  return (
    <>
      <Flex align='center' gap={12} wrap>
        <ProjectListFilter
          initialValue={filter}
          onChange={handleFilterChange}
          className='max-w-[480px] flex-[1_1_240px]'
        />
        {!loading && (
          <Typography.Text type='secondary' className='text-xs'>
            {total} project{total === 1 ? '' : 's'}
          </Typography.Text>
        )}
        {/* AntD draws the track in the page's own color, which loses it on a
            light page. Dark sets its own track in the theme config. */}
        <Segmented<View>
          className="ml-auto [[data-theme='light']_&]:!bg-[color:var(--kargo-color-fill-secondary)]"
          value={view}
          onChange={handleViewChange}
          options={[
            { value: 'all', label: 'All' },
            {
              value: 'mine',
              icon: <FontAwesomeIcon icon={faUser} />,
              label: (
                <Tooltip title={MINE_TOOLTIP}>
                  <span>Mine</span>
                </Tooltip>
              )
            },
            { value: 'starred', icon: <FontAwesomeIcon icon={faStar} />, label: 'Starred' }
          ]}
        />
      </Flex>
      <Divider className='!mb-5 !mt-4' />

      {loading ? (
        <div
          role='status'
          aria-label='Loading projects'
          className={classNames(gridClassName, 'motion-safe:animate-pulse')}
        >
          {Array.from({ length: SKELETON_COUNT }, (_, i) => (
            <ProjectItemSkeleton key={i} />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <Empty
          className={classNames('py-12 transition-opacity', {
            // With nothing starred there is no request, but the query still
            // holds the previous view's results as placeholder data.
            'opacity-60': isPlaceholderData && !noStars
          })}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={emptyDescription()}
        />
      ) : (
        <>
          {/* The previous results dim while the next ones load. */}
          <div
            className={classNames(gridClassName, 'transition-opacity', {
              'opacity-60': isPlaceholderData
            })}
          >
            {projects.map((proj) => (
              <ProjectItem
                key={proj?.metadata?.name}
                project={proj}
                starred={starred.includes(proj?.metadata?.uid || '')}
                onToggleStar={toggleStar}
              />
            ))}
          </div>
          <Flex justify='flex-end' className='mt-6'>
            <Pagination
              total={total}
              pageSize={pageSize}
              pageSizeOptions={PAGE_SIZE_OPTIONS}
              current={page}
              onChange={(newPage, newPageSize) => {
                setPage(newPage);
                setPageSize(newPageSize);
              }}
              showSizeChanger
              hideOnSinglePage
            />
          </Flex>
        </>
      )}
    </>
  );
};
