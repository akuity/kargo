import { useQuery } from '@connectrpc/connect-query';
import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt, faGithub } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faCaretLeft,
  faCaretRight,
  faChevronLeft,
  faChevronRight,
  faExternalLink,
  faFilter,
  faTimes,
  faWarehouse,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  Button,
  Checkbox,
  Descriptions,
  Divider,
  Select,
  SelectProps,
  Table,
  Tag,
  Typography
} from 'antd';
import Link from 'antd/es/typography/Link';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { ReactNode, useMemo, useRef, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import { LoadingState } from '@ui/features/common';
import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { flattenFreightOrigin } from '@ui/features/freight/flatten-freight-origin-utils';
import { FreightStatusList } from '@ui/features/freight/freight-status-list';
import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import {
  listStages,
  listWarehouses,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Chart, Freight, GitCommit, Image, Project } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import './pipelines.less';

import { humanComprehendableArtifact } from './artifact-parts-utils';
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
import {
  timerangeOrderedOptions,
  timerangeToDate,
  timerangeToLabel
} from './filter-timerange-utils';
import { Graph } from './graph/graph';
import { shortVersion } from './short-version-utils';
import {
  catalogueFreights,
  filterFreightBySource,
  filterFreightByTimerange
} from './source-catalogue-utils';

import '@xyflow/react/dist/style.css';

export const Pipelines = (props: { project: Project }) => {
  const projectName = props.project?.metadata?.name;

  const getFreightQuery = useQuery(queryFreight, { project: projectName });

  const listWarehousesQuery = useQuery(listWarehouses, { project: projectName });

  const listStagesQuery = useQuery(listStages, { project: projectName });

  const loading =
    getFreightQuery.isLoading || listWarehousesQuery.isLoading || listStagesQuery.isLoading;

  if (loading) {
    return <LoadingState />;
  }

  return (
    <>
      <ColorContext.Provider value={{ stageColorMap: {}, warehouseColorMap: {} }}>
        <FreightTimeline freights={getFreightQuery?.data?.groups?.['']?.freight || []} />

        <div className='w-full h-full'>
          <Graph
            project={props.project.metadata?.name || ''}
            warehouses={listWarehousesQuery.data?.warehouses || []}
            stages={listStagesQuery.data?.stages || []}
          />
        </div>
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

  const sourcesDropdownOptions: SelectProps['options'] = useMemo(() => {
    const freightSourcesCatalogue = catalogueFreights(props.freights);

    const opts: SelectProps['options'] = [];

    for (const [sourceType, repoURLs] of Object.entries(freightSourcesCatalogue)) {
      let icon: IconDefinition = faGitAlt;

      if (sourceType === 'images') {
        icon = faDocker;
      } else if (sourceType === 'charts') {
        icon = faAnchor;
      }

      for (const repoURL of repoURLs) {
        opts.push({
          value: repoURL,
          label: (
            <div className='w-fit'>
              <FontAwesomeIcon icon={icon} className='mr-2' />
              <span className='text-xs'>{repoURL}</span>
            </div>
          )
        });
      }
    }

    return opts;
  }, [props.freights]);

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
                value={preferredFilter?.sources}
                dropdownStyle={{ width: '50%' }}
                onChange={(sources) => setPreferredFilter({ ...preferredFilter, sources })}
                labelRender={(props) => humanComprehendableArtifact(props.value.toString())}
                placeholder='All'
                options={sourcesDropdownOptions}
                maxTagCount={1}
              />
            </div>
            <div className='text-xs flex items-center gap-3 mt-2'>
              <label>Timerange: </label>
              <Select
                className='min-w-[200px]'
                size='small'
                value={preferredFilter?.timerange}
                options={timerangeOrderedOptions.map((opt) => ({
                  value: opt,
                  label: <>{timerangeToLabel(opt)}</>
                }))}
                maxTagCount={1}
                onChange={(timerange) =>
                  setPreferredFilter({ ...preferredFilter, timerange: timerange })
                }
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
                Alias
              </Checkbox>

              <Checkbox
                className='text-xs'
                checked={!preferredFilter?.artifactCarousel?.enabled}
                onChange={(e) => {
                  const enabled = !e.target.checked;

                  if (enabled) {
                    const firstArtifact = selectFirstArtifact(filteredFreights);

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
                All Artifacts
              </Checkbox>

              <Checkbox
                className='text-xs'
                checked={preferredFilter?.showColors}
                onChange={(e) =>
                  setPreferredFilter({ ...preferredFilter, showColors: e.target.checked })
                }
              >
                Colors
              </Checkbox>
            </div>
          </div>
        </div>
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
              <FreightCard key={freight?.metadata?.uid} freight={freight} />
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
      <Tag
        className='w-fit text-[8px] absolute right-[-20px] top-0 leading-none'
        bordered={false}
        color='green'
      >
        in use
      </Tag>
      {freightTimelineControllerContext?.preferredFilter?.showColors && (
        <div className='flex gap-1 mb-1 justify-center'>
          <div
            title='dev'
            className='mx-1 h-3 w-3 rounded'
            style={{
              background: '#45B084'
            }}
          />
        </div>
      )}

      {freightTimelineControllerContext?.preferredFilter?.showAlias && (
        <div className='text-[10px] text-nowrap mb-2'>{freightAlias}</div>
      )}

      {!freightTimelineControllerContext?.preferredFilter?.artifactCarousel?.enabled && (
        <div className='flex gap-1 justify-center'>
          {props.freight?.commits?.map((commit) => (
            <FreightCard.Artifact key={commit?.repoURL} artifact={commit} />
          ))}

          {props.freight?.charts?.map((chart) => (
            <FreightCard.Artifact key={chart?.repoURL} artifact={chart} />
          ))}

          {props.freight?.images?.map((image) => (
            <FreightCard.Artifact key={image?.repoURL} artifact={image} />
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
        <Typography.Text className='text-[10px] text-nowrap' type='secondary'>
          <FontAwesomeIcon className='mr-1' icon={faWarehouse} />
          {props.freight?.origin?.name}
        </Typography.Text>
      </div>
    </div>
  );
};

FreightCard.Artifact = (props: { artifact: GitCommit | Chart | Image; expand?: boolean }) => {
  const artifactType = props.artifact?.$typeName;

  let Expand: ReactNode;

  if (props.expand) {
    Expand = (
      <span className='text-[10px] ml-1'>
        {humanComprehendableArtifact(props.artifact.repoURL)}
      </span>
    );
  }

  if (artifactType === 'github.com.akuity.kargo.api.v1alpha1.GitCommit') {
    const url = getGitCommitURL(props.artifact.repoURL, props.artifact.id);

    const TagComponent = (
      <Tag title={props.artifact.repoURL} bordered={false} color='geekblue' key={props.artifact.id}>
        {props.artifact.id.slice(0, 7)}

        {!!url && (
          <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 text-[8px] ml-1' />
        )}

        {Expand}
      </Tag>
    );

    if (url) {
      return (
        <Link
          key={props.artifact.repoURL}
          href={url}
          target='_blank'
          onClick={(e) => e.stopPropagation()}
        >
          {TagComponent}
        </Link>
      );
    }

    return TagComponent;
  }

  if (artifactType === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
    return (
      <Tag
        title={`${props.artifact.repoURL}:${props.artifact.version}`}
        bordered={false}
        color='geekblue'
        key={props.artifact.repoURL}
      >
        {shortVersion(props.artifact.version)}

        {Expand}
      </Tag>
    );
  }

  let imageSourceFromOci = '';

  if (props.artifact.annotations) {
    imageSourceFromOci = getImageSource(props.artifact.annotations);
  }

  const TagComponent = (
    <Tag
      title={`${props.artifact.repoURL}:${props.artifact.tag}`}
      bordered={false}
      color='geekblue'
      key={props.artifact?.repoURL}
    >
      {shortVersion(props.artifact?.tag)}

      {!!imageSourceFromOci && (
        <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 ml-1 text-[8px]' />
      )}

      {Expand}
    </Tag>
  );

  if (imageSourceFromOci) {
    return (
      <Link
        key={props.artifact?.repoURL}
        href={imageSourceFromOci}
        target='_blank'
        onClick={(e) => e.stopPropagation()}
      >
        {TagComponent}
      </Link>
    );
  }

  return TagComponent;
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

      <FreightCard.Artifact artifact={activeArtifact} expand />

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
