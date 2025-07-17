/* eslint-disable @typescript-eslint/no-explicit-any */
import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faBook, faEyeSlash, faChevronUp } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo, useContext, useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';
import semver from 'semver';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { TagMap } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

const findWarehousesForImageRepo = (repoURL: string, warehouses: Warehouse[]): string[] => {
  return warehouses
    .filter((warehouse) =>
      warehouse.spec?.subscriptions?.some((sub) => sub.image?.repoURL === repoURL)
    )
    .map((warehouse) => warehouse.metadata?.name || '')
    .filter(Boolean);
};

const findStagesForWarehouse = (warehouseName: string, stages: Stage[]): Set<string> => {
  const reachableStages = new Set<string>();
  stages.forEach((stage) => {
    const stageName = stage.metadata?.name;
    if (!stageName) return;
    const requestsFromWarehouse = stage.spec?.requestedFreight?.some(
      (req) => req.origin?.name === warehouseName
    );
    if (requestsFromWarehouse) {
      reachableStages.add(stageName);
    }
  });
  return reachableStages;
};

const getStagesForImage = (
  repoURL: string,
  warehouses: Warehouse[],
  stages: Stage[]
): Set<string> => {
  const warehousesForImage = findWarehousesForImageRepo(repoURL, warehouses);
  const allReachableStages = new Set<string>();
  warehousesForImage.forEach((warehouseName) => {
    const stagesForWarehouse = findStagesForWarehouse(warehouseName, stages);
    stagesForWarehouse.forEach((stageName) => allReachableStages.add(stageName));
  });
  return allReachableStages;
};

interface ImagesProps {
  hide: () => void;
  images: Record<string, TagMap>;
  project: string;
  stages: Stage[];
  warehouses: Warehouse[];
}

const ImagesCompactCard = memo<ImagesProps>(({ hide, images, project, stages, warehouses }) => {
  const { stageColorMap } = useContext(ColorContext);
  const [showHistory, setShowHistory] = useLocalStorage('images-show-history', false);

  const filteredImages = useMemo(() => {
    const result: Record<string, any> = {};
    Object.entries(images).forEach(([repoURL, tagMap]) => {
      const stagesForImage = getStagesForImage(repoURL, warehouses, stages);
      const filteredTagMap = { tags: {} as Record<string, any> };
      Object.entries(tagMap.tags || {}).forEach(([tag, imageStageMap]) => {
        const filteredStages: Record<string, number> = {};
        Object.entries(imageStageMap.stages || {}).forEach(([stageName, order]) => {
          if (stagesForImage.has(stageName)) {
            filteredStages[stageName] = order;
          }
        });
        if (Object.keys(filteredStages).length > 0) {
          const newImageStageMap = { ...imageStageMap, stages: filteredStages };
          filteredTagMap.tags[tag] = newImageStageMap;
        }
      });
      if (Object.keys(filteredTagMap.tags || {}).length > 0) {
        result[repoURL] = filteredTagMap;
      }
    });
    return result;
  }, [images, warehouses, stages]);

  const repoURLs = useMemo(() => Object.keys(filteredImages), [filteredImages]);
  const [selectedRepoURL, setSelectedRepoURL] = useState<string>('');

  useEffect(() => {
    if (repoURLs.length > 0 && !repoURLs.includes(selectedRepoURL)) {
      setSelectedRepoURL(repoURLs[0]);
    } else if (repoURLs.length === 0) {
      setSelectedRepoURL('');
    }
  }, [repoURLs, selectedRepoURL]);

  const imagesToRender = useMemo(() => {
    if (!selectedRepoURL) {
      return [];
    }
    const singleImage = filteredImages[selectedRepoURL];
    return singleImage ? [[selectedRepoURL, singleImage]] : [];
  }, [selectedRepoURL, filteredImages]);

  if (repoURLs.length === 0) {
    return (
      <div className='bg-white rounded-lg p-3'>
        <div className='flex items-center justify-between mb-3 px-1'>
          <h3 className='text-base font-semibold'>Images</h3>
          <HeaderButton onClick={hide} icon={faEyeSlash} title='Hide panel' />
        </div>
        <div className='text-gray-500 text-center py-8'>No images found</div>
      </div>
    );
  }

  return (
    <div className='bg-white rounded-lg p-3 flex flex-col max-h-[calc(100vh-120px)]'>
      <div className='flex-shrink-0'>
        <div className='flex items-center justify-between mb-3 px-1'>
          <h3 className='text-base font-semibold'>Images</h3>
          <div className='flex items-center gap-2'>
            <HeaderButton
              onClick={() => setShowHistory(!showHistory)}
              selected={showHistory}
              icon={faBook}
              title={showHistory ? 'Hide promotion history' : 'Show promotion history'}
            />
            <HeaderButton onClick={hide} icon={faEyeSlash} title='Hide panel' />
          </div>
        </div>

        {repoURLs.length > 1 && (
          <div className='mb-3'>
            <select
              value={selectedRepoURL}
              onChange={(e) => setSelectedRepoURL(e.target.value)}
              className='block border-gray-300 w-full text-gray-800 appearance-none p-2 bg-white rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500'
            >
              {repoURLs.map((repo) => (
                <option value={repo} key={repo}>
                  {repo.split('/').pop() || repo}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      <div className='overflow-y-auto'>
        <div className='flex flex-col gap-3'>
          {imagesToRender.map(([repoURL, tagMap]) => (
            <ImageCard
              key={repoURL}
              repoURL={repoURL}
              tagMap={tagMap}
              project={project}
              stageColorMap={stageColorMap}
              showHistory={showHistory}
            />
          ))}
        </div>
      </div>
    </div>
  );
});
ImagesCompactCard.displayName = 'Images';

const HeaderButton = memo(
  ({
    onClick,
    icon,
    selected,
    title
  }: {
    onClick: () => void;
    icon: IconDefinition;
    selected?: boolean;
    title: string;
  }) => (
    <Tooltip title={title}>
      <button
        onClick={onClick}
        className={classNames(
          'p-2 w-7 h-7 flex items-center justify-center rounded-md hover:bg-gray-100 transition-colors',
          selected ? 'bg-blue-100 text-blue-600' : 'text-gray-400 hover:text-gray-600'
        )}
      >
        <FontAwesomeIcon icon={icon} />
      </button>
    </Tooltip>
  )
);
HeaderButton.displayName = 'HeaderButton';

const ImageCard = memo(
  ({
    repoURL,
    tagMap,
    project,
    stageColorMap,
    showHistory
  }: {
    repoURL: string;
    tagMap: TagMap;
    project: string;
    stageColorMap: Record<string, string>;
    showHistory: boolean;
  }) => {
    const sortedTags = useMemo(() => {
      return Object.keys(tagMap.tags || {}).sort((a, b) => {
        try {
          return semver.compare(b, a);
        } catch {
          return b.localeCompare(a);
        }
      });
    }, [tagMap.tags]);

    return (
      <div className='bg-gray-50/70 rounded-lg p-2.5'>
        <div className='flex items-center gap-2 text-gray-600 px-1 mb-2'>
          <FontAwesomeIcon icon={faDocker} className='text-blue-500' />
          <span className='font-semibold text-sm truncate'>{repoURL.split('/').pop()}</span>
        </div>
        <div className='flex flex-col'>
          {sortedTags.map((tag) => {
            const imageStageMap = tagMap.tags[tag];
            if (!imageStageMap?.stages) return null;

            const stagesWithImage = Object.entries(imageStageMap.stages || {})
              .sort(([, a], [, b]) => (a as number) - (b as number))
              .map(([stageName, order]) => ({ stageName, order: order as number }));

            if (stagesWithImage.length === 0) return null;

            return (
              <ImageVersionRow
                key={tag}
                tag={tag}
                stagesWithImage={stagesWithImage}
                project={project}
                stageColorMap={stageColorMap}
                showHistory={showHistory}
              />
            );
          })}
        </div>
      </div>
    );
  }
);
ImageCard.displayName = 'ImageCard';

const ImageVersionRow = memo(
  ({
    tag,
    stagesWithImage,
    project,
    stageColorMap,
    showHistory
  }: {
    tag: string;
    stagesWithImage: Array<{ stageName: string; order: number }>;
    project: string;
    stageColorMap: Record<string, string>;
    showHistory: boolean;
  }) => {
    const navigate = useNavigate();
    const [isExpanded, setIsExpanded] = useState(false);

    const MAX_VISIBLE_STAGES = 3;
    const hasMoreStages = stagesWithImage.length > MAX_VISIBLE_STAGES + 1;
    const hiddenStageCount = stagesWithImage.length - MAX_VISIBLE_STAGES;

    const visibleStages = isExpanded
      ? stagesWithImage
      : stagesWithImage.slice(
          0,
          hiddenStageCount === 1 ? MAX_VISIBLE_STAGES + 1 : MAX_VISIBLE_STAGES
        );

    const handleStageClick = (stageName: string) => {
      navigate(generatePath(paths.stage, { name: project, stageName }));
    };

    return (
      <div className='bg-white rounded p-2.5 flex items-start gap-3'>
        <div className='w-1/6 flex-shrink-0'>
          <Tooltip title={`${stagesWithImage.length} Stage(s)`}>
            <div className='font-mono text-sm font-semibold text-gray-800 w-full truncate'>
              {tag}
            </div>
          </Tooltip>
        </div>
        <div className='flex-grow min-w-0'>
          <div className='grid grid-cols-4 gap-1.5'>
            {visibleStages.map(({ stageName, order }) => {
              const isHighlighted = showHistory && order === 0;
              const baseColor = stageColorMap[stageName] || '#6b7280';
              const tooltipTitle = showHistory
                ? isHighlighted
                  ? `Most recent promotion: Currently in stage '${stageName}'`
                  : `Stage: '${stageName}' (Promotion #${order + 1})`
                : `Promoted to stage: '${stageName}'`;

              const style: React.CSSProperties = {
                borderColor: baseColor,
                backgroundColor: `${baseColor}10`,
                transition: 'all 0.2s ease-in-out'
              };

              if (showHistory) {
                if (isHighlighted) {
                  style.opacity = 1;
                  style.boxShadow = `0 0 0 2px ${baseColor}`;
                } else {
                  style.opacity = 0.6;
                }
              }

              return (
                <Tooltip key={stageName} title={tooltipTitle}>
                  <button
                    onClick={() => handleStageClick(stageName)}
                    className='w-full px-2 py-0.5 rounded border text-xs font-medium'
                    style={style}
                  >
                    <div className='flex items-center justify-between'>
                      <span
                        className='truncate'
                        style={{ color: stageColorMap[stageName] || '#6b7280' }}
                      >
                        {stageName}
                      </span>
                      {showHistory && (
                        <span
                          className='ml-1.5 text-white font-mono text-xs px-1 rounded-full'
                          style={{ backgroundColor: stageColorMap[stageName] || '#6b7280' }}
                        >
                          {order + 1}
                        </span>
                      )}
                    </div>
                  </button>
                </Tooltip>
              );
            })}
            {hasMoreStages && !isExpanded && (
              <button
                onClick={() => setIsExpanded(true)}
                className='w-full px-2 py-0.5 rounded border text-xs font-medium text-gray-500 bg-gray-100 hover:bg-gray-200 border-dashed'
              >
                +{hiddenStageCount} More
              </button>
            )}
          </div>
          {hasMoreStages && isExpanded && (
            <div className='mt-1.5'>
              <button
                onClick={() => setIsExpanded(false)}
                className='w-full text-xs font-medium text-blue-600 hover:text-blue-800 flex items-center justify-center gap-1 py-0.5 rounded-md hover:bg-blue-50'
              >
                <FontAwesomeIcon icon={faChevronUp} className='text-xs' />
                Show Less
              </button>
            </div>
          )}
        </div>
      </div>
    );
  }
);
ImageVersionRow.displayName = 'ImageVersionRow';

export { ImagesCompactCard };
