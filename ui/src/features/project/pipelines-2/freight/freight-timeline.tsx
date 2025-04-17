import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faCaretLeft, faCaretRight, faTimes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Descriptions, Divider, Table } from 'antd';
import classNames from 'classnames';
import { useMemo, useRef, useState } from 'react';

import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { flattenFreightOrigin } from '@ui/features/freight/flatten-freight-origin-utils';
import { FreightStatusList } from '@ui/features/freight/freight-status-list';
import { useDictionaryContext } from '@ui/features/project/pipelines-2/context/dictionary-context';
import {
  FreightTimelineControllerContext,
  FreightTimelineControllerContextType
} from '@ui/features/project/pipelines-2/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { timerangeToDate } from './filter-timerange-utils';
import { FreightCard } from './freight-card';
import { FreightTimelineFilters } from './freight-timeline-filters';
import { filterFreightBySource, filterFreightByTimerange } from './source-catalogue-utils';

import './freight-timeline.less';

export const FreightTimeline = (props: { freights: Freight[] }) => {
  const dictionaryContext = useDictionaryContext();

  const [filtersCollapsed, setFilterCollapsed] = useState(true);

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const [preferredFilter, setPreferredFilter] = useState<
    FreightTimelineControllerContextType['preferredFilter']
  >({
    showAlias: false,
    artifactCarousel: {
      enabled: false
    },
    sources: [],
    timerange: 'all-time',
    showColors: false
  });

  const filteredFreights = useMemo(() => {
    let filtered = props.freights?.sort((a, b) => {
      const t1 = timestampDate(a?.metadata?.creationTimestamp);

      const t2 = timestampDate(b?.metadata?.creationTimestamp);

      return (t2?.getTime() || 0) - (t1?.getTime() || 0);
    });

    filtered = filtered
      .map(filterFreightBySource(preferredFilter?.sources))
      .filter(Boolean) as Freight[];

    if (preferredFilter.timerange !== 'all-time') {
      filtered = filtered.filter(
        filterFreightByTimerange(timerangeToDate(preferredFilter.timerange))
      );
    }

    return filtered;
  }, [props.freights, preferredFilter.sources, preferredFilter.timerange]);

  const freightListStyleRef = useRef<HTMLDivElement>(null);

  const scrollCarouselLeft = () => {
    const right = freightListStyleRef.current?.style.right || '0px';

    let nextRight = +right.slice(0, -2) - 80;

    if (nextRight < 0) {
      nextRight = 0;
    }

    freightListStyleRef.current?.style.setProperty('right', `${nextRight}px`);
  };

  const scrollCarouselRight = () => {
    const right = freightListStyleRef.current?.style.right || '0px';

    const nextRight = +right.slice(0, -2) + 80;

    if (nextRight >= (freightListStyleRef.current?.clientWidth || 0) / 2) {
      return;
    }

    freightListStyleRef.current?.style.setProperty('right', `${nextRight}px`);
  };

  return (
    <FreightTimelineControllerContext.Provider
      value={{
        viewingFreight,
        setViewingFreight,
        preferredFilter,
        setPreferredFilter
      }}
    >
      <div
        className={classNames('freightTimeline', 'bg-white px-5 py-2 flex gap-5')}
        style={{ borderBottom: '2px solid rgba(0,0,0,.05)' }}
      >
        <FreightTimelineFilters
          collapsed={filtersCollapsed}
          filteredFreights={filteredFreights}
          freights={props.freights}
          onCollapseToggle={() => setFilterCollapsed(!filtersCollapsed)}
          onPreferredFilterChange={setPreferredFilter}
          preferredFilter={preferredFilter}
        />
        {!filtersCollapsed && <Divider type='vertical' className='h-full' />}
        <div
          className='w-full flex overflow-hidden relative px-5'
          onWheel={(e) => {
            if (e.deltaX > 0) {
              scrollCarouselRight();
              return;
            }

            scrollCarouselLeft();
          }}
        >
          <div className='flex gap-1 relative transition-all right-0' ref={freightListStyleRef}>
            {filteredFreights.map((freight) => (
              <FreightCard
                key={freight?.metadata?.uid}
                freight={freight}
                preferredFilter={preferredFilter}
                setViewingFreight={setViewingFreight}
                viewingFreight={viewingFreight}
                inUse={
                  (dictionaryContext?.freightInStages[freight?.metadata?.name || '']?.length || 0) >
                  0
                }
              />
            ))}
          </div>

          <div
            className='absolute left-0 h-full bg-gray-100 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselLeft();
            }}
          >
            <FontAwesomeIcon icon={faCaretLeft} />
          </div>

          <div
            className='absolute right-0 h-full bg-gray-100 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselRight();
            }}
          >
            <FontAwesomeIcon icon={faCaretRight} />
          </div>
        </div>
      </div>

      {!!viewingFreight && (
        <div className='scale-90 origin-top bg-white p-5'>
          <FreightExtended freight={viewingFreight} onClose={() => setViewingFreight(null)} />
        </div>
      )}
    </FreightTimelineControllerContext.Provider>
  );
};

const FreightExtended = (props: { freight: Freight; onClose?: () => void }) => {
  return (
    <div className='relative'>
      <Button
        icon={<FontAwesomeIcon icon={faTimes} />}
        type='text'
        size='large'
        className='absolute right-2'
        onClick={() => props.onClose?.()}
      />
      <Descriptions
        column={1}
        size='small'
        items={[
          {
            label: 'Warehouse',
            children: props.freight?.origin?.name
          },
          {
            label: 'Name',
            children: props.freight?.metadata?.name
          },
          {
            label: 'UID',
            children: props.freight?.metadata?.uid
          },
          {
            label: 'Creation Timestamp',
            children: timestampDate(props.freight?.metadata?.creationTimestamp)?.toString()
          }
        ]}
      />

      <Table
        className='mt-5'
        pagination={{
          pageSize: 5
        }}
        dataSource={flattenFreightOrigin(props.freight)}
        columns={[
          {
            title: 'Source',
            render: (_, { type }) => {
              let icon: IconProp = faGitAlt;

              switch (type) {
                case 'helm':
                  icon = faAnchor;
                  break;
                case 'image':
                  icon = faDocker;
                  break;
              }

              return <FontAwesomeIcon icon={icon} />;
            },
            width: '5%'
          },
          {
            title: 'Repo',
            dataIndex: 'repoURL',
            width: '30%'
          },
          {
            title: 'Version',
            render: (_, record) => {
              switch (record.type) {
                case 'git':
                  return record.id;
                case 'helm':
                  return record.version;
                case 'image':
                  return record.tag;
              }
            }
          },
          {
            title: 'Metadata',
            width: '600px',
            render: (_, record) => {
              return <ArtifactMetadata {...record} />;
            }
          }
        ]}
      />
      <FreightStatusList freight={props.freight} />
    </div>
  );
};
