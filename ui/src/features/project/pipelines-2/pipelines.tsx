import { useQuery } from '@connectrpc/connect-query';
import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt, faGithub } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faCaretLeft,
  faCaretRight,
  faChevronLeft,
  faChevronRight,
  faFilter,
  faTimes,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Checkbox, Descriptions, Divider, Select, Table, Tag, Typography } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { useMemo, useRef, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { flattenFreightOrigin } from '@ui/features/freight/flatten-freight-origin-utils';
import { FreightStatusList } from '@ui/features/freight/freight-status-list';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Project } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import './pipelines.less';
import {
  selectActiveCarouselFreight,
  selectFirstArtifact,
  selectNextArtifact,
  selectPreviousArtifact
} from './artifact-selector-utils';
import {
  FreightTimelineControllerContext,
  FreightTimelineControllerContextType,
  useFreightTimelineControllerContext
} from './context/freight-timeline-controller-context';
import { shortVersion } from './short-version-utils';

export const Pipelines = (props: { project: Project }) => {
  const getFreightQuery = useQuery(queryFreight, { project: props.project?.metadata?.name });

  const loading = getFreightQuery.isLoading;

  if (loading) {
    return <LoadingState />;
  }

  return (
    <>
      <ColorContext.Provider value={{ stageColorMap: {}, warehouseColorMap: {} }}>
        <FreightTimeline freights={getFreightQuery?.data?.groups?.['']?.freight || []} />
      </ColorContext.Provider>
    </>
  );
};

const FreightTimeline = (props: { freights: Freight[] }) => {
  const [filtersCollapsed, setFilterCollapsed] = useState(true);

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const [preferredFilter, setPreferredFilter] = useState<
    FreightTimelineControllerContextType['preferredFilter']
  >({
    showAlias: false,
    artifactCarousel: {
      enabled: false
    }
  });

  const orderedFreights = useMemo(() => props.freights?.reverse(), [props.freights]);

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
        <div>
          <span className='text-xs flex items-center gap-2'>
            {!filtersCollapsed && (
              <div className='font-semibold flex items-center gap-2'>
                <FontAwesomeIcon icon={faFilter} /> Filters
              </div>
            )}

            <Button
              size='small'
              className='ml-auto'
              onClick={() => setFilterCollapsed(!filtersCollapsed)}
            >
              <FontAwesomeIcon icon={filtersCollapsed ? faFilter : faTimes} />
            </Button>
          </span>

          <div
            className={classNames('transition-all', {
              'w-0 h-0 opacity-0': filtersCollapsed,
              'w-full': !filtersCollapsed
            })}
          >
            <div className={'text-xs flex items-center gap-3 mt-2'}>
              <label>Source: </label>
              <Select
                mode='multiple'
                className='min-w-[200px] ml-auto'
                size='small'
                defaultValue={['foo', 'bar', 'baz']}
                maxTagCount={1}
              />
            </div>
            <div className='text-xs flex items-center gap-3 mt-2'>
              <label>Timerange: </label>
              <Select
                className='min-w-[200px]'
                size='small'
                defaultValue={['7 Days']}
                maxTagCount={1}
              />
            </div>

            <div className='flex mt-3 gap-2'>
              <Checkbox
                className='text-xs'
                checked={preferredFilter?.showAlias}
                onChange={(e) =>
                  setPreferredFilter({ ...preferredFilter, showAlias: e.target.checked })
                }
              >
                Show Alias
              </Checkbox>

              <Checkbox
                className='text-xs'
                checked={!preferredFilter?.artifactCarousel?.enabled}
                onChange={(e) => {
                  const enabled = !e.target.checked;

                  if (enabled) {
                    const firstArtifact = selectFirstArtifact(orderedFreights);

                    setPreferredFilter({
                      ...preferredFilter,
                      artifactCarousel: {
                        enabled,
                        state: {
                          repoURL: typeof firstArtifact === 'string' ? '' : firstArtifact?.repoURL
                        }
                      }
                    });
                    return;
                  }

                  setPreferredFilter({ ...preferredFilter, artifactCarousel: { enabled } });
                }}
              >
                Show all Artifacts
              </Checkbox>
            </div>
          </div>
        </div>
        {!filtersCollapsed && <Divider type='vertical' className='h-full' />}
        <div className='w-full flex overflow-hidden relative px-5'>
          <div className='flex gap-1 relative transition-all right-0' ref={freightListStyleRef}>
            {orderedFreights.map((freight) => (
              <FreightCard key={freight?.metadata?.name} freight={freight} />
            ))}
          </div>

          <div
            className='absolute left-0 h-full bg-gray-200 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselLeft();
            }}
          >
            <FontAwesomeIcon icon={faCaretLeft} />
          </div>

          <div
            className='absolute right-0 h-full bg-gray-300 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselRight();
            }}
          >
            <FontAwesomeIcon icon={faCaretRight} />
          </div>
        </div>
      </div>

      {!!viewingFreight && (
        <div className='scale-75 origin-top bg-white p-5'>
          <FreightExtended freight={viewingFreight} onClose={() => setViewingFreight(null)} />
        </div>
      )}
    </FreightTimelineControllerContext.Provider>
  );
};

const FreightCard = (props: { freight: Freight }) => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();
  const freightAlias = props.freight?.alias;

  const creation = useMemo(() => {
    const creationDate = timestampDate(props.freight?.metadata?.creationTimestamp);

    if (!creationDate) {
      return { relative: '', abs: creationDate };
    }

    return {
      relative: formatDistance(creationDate, new Date(), { addSuffix: false })?.replace(
        'about',
        ''
      ),
      abs: creationDate
    };
  }, [props.freight]);

  const noOfGitCommits = props.freight?.commits?.length || 0;
  const noOfHelmReleases = props.freight?.charts?.length || 0;
  const noOfContainerImages = props.freight?.images?.length || 0;

  const isViewingFreight =
    freightTimelineControllerContext?.viewingFreight?.metadata?.name ===
    props.freight?.metadata?.name;

  return (
    <div
      className={classNames(
        'pt-2 px-2 rounded-md text-center flex flex-col cursor-pointer hover:bg-gray-100 relative justify-center',
        {
          'bg-gray-50': !isViewingFreight,
          'bg-gray-100': isViewingFreight
        }
      )}
      style={{ border: '1px solid rgba(0,0,0,.05)' }}
      onClick={() =>
        freightTimelineControllerContext?.setViewingFreight(isViewingFreight ? null : props.freight)
      }
    >
      {freightTimelineControllerContext?.preferredFilter?.showAlias && (
        <div className='text-[10px] text-nowrap mb-2'>{freightAlias}</div>
      )}

      {!freightTimelineControllerContext?.preferredFilter?.artifactCarousel?.enabled && (
        <div className='flex gap-1 justify-center mb-auto'>
          {props.freight?.commits?.map((commit) => (
            <Tag title={commit?.repoURL} bordered={false} color='blue' key={commit?.id}>
              {commit?.id?.slice(0, 7)}
            </Tag>
          ))}

          {props.freight?.charts?.map((chart) => (
            <Tag
              title={`${chart.repoURL}:${chart.version}`}
              bordered={false}
              color='blue'
              key={chart.repoURL}
            >
              {shortVersion(chart?.version)}
            </Tag>
          ))}

          {props.freight?.images?.map((image) => (
            <Tag
              title={`${image.repoURL}:${image.tag}`}
              bordered={false}
              color='blue'
              key={image?.repoURL}
            >
              {shortVersion(image?.tag)}
            </Tag>
          ))}
        </div>
      )}

      {freightTimelineControllerContext?.preferredFilter?.artifactCarousel?.enabled && (
        <FreightCard.ArtifactCarousel freight={props.freight} />
      )}

      <div className='flex mx-auto w-full gap-2 items-center justify-center text-nowrap mt-1'>
        {noOfGitCommits + noOfHelmReleases + noOfContainerImages > 0 && (
          <>
            <FreightCard.ArtifactCount icon={faGithub} count={noOfGitCommits} />

            <FreightCard.ArtifactCount icon={faAnchor} count={noOfHelmReleases} />

            <FreightCard.ArtifactCount icon={faDocker} count={noOfContainerImages} />
          </>
        )}
        <Typography.Text
          className='text-[10px] text-nowrap'
          type='secondary'
          title={creation.abs?.toString()}
        >
          {creation.relative}
        </Typography.Text>
      </div>
    </div>
  );
};

FreightCard.ArtifactCarousel = (props: { freight: Freight }) => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const activeArtifact = selectActiveCarouselFreight(
    props.freight,
    freightTimelineControllerContext?.preferredFilter?.artifactCarousel?.state?.repoURL || ''
  );

  if (typeof activeArtifact === 'string') {
    return 'Invalid Artifact';
  }

  return (
    <div className='flex gap-1 mb-1 items-center justify-between'>
      <Button
        type='text'
        icon={<FontAwesomeIcon icon={faChevronLeft} />}
        onClick={(e) => {
          e.stopPropagation();
          const previousArtifact = selectPreviousArtifact(
            props.freight,
            activeArtifact?.repoURL || ''
          );

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            artifactCarousel: {
              ...freightTimelineControllerContext?.preferredFilter?.artifactCarousel,
              state: {
                repoURL: previousArtifact
              }
            }
          });
        }}
      />
      {activeArtifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.GitCommit' && (
        <Tag title={activeArtifact?.repoURL} bordered={false} color='blue' key={activeArtifact?.id}>
          {activeArtifact?.id?.slice(0, 7)}
        </Tag>
      )}

      {activeArtifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Chart' && (
        <Tag
          title={`${activeArtifact.repoURL}:${activeArtifact.version}`}
          bordered={false}
          color='blue'
          key={activeArtifact.repoURL}
        >
          {shortVersion(activeArtifact?.version)}
        </Tag>
      )}

      {activeArtifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Image' && (
        <Tag
          title={`${activeArtifact.repoURL}:${activeArtifact.tag}`}
          bordered={false}
          color='blue'
          key={activeArtifact?.repoURL}
        >
          {shortVersion(activeArtifact?.tag)}
        </Tag>
      )}

      <Button
        type='text'
        icon={<FontAwesomeIcon icon={faChevronRight} />}
        onClick={(e) => {
          e.stopPropagation();
          const nextArtifact = selectNextArtifact(props.freight, activeArtifact?.repoURL || '');

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            artifactCarousel: {
              ...freightTimelineControllerContext?.preferredFilter?.artifactCarousel,
              state: {
                repoURL: nextArtifact
              }
            }
          });
        }}
      />
    </div>
  );
};

FreightCard.ArtifactCount = (props: { icon: IconDefinition; count: number }) =>
  props.count > 0 && (
    <div className='text-[10px]'>
      {props.count}x <FontAwesomeIcon icon={props.icon} />
    </div>
  );

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
