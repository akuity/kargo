import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faBook, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
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
    .filter((w) => w.spec?.subscriptions?.some((s) => s.image?.repoURL === repoURL))
    .map((w) => w.metadata?.name || '')
    .filter(Boolean);
};
const findStagesForWarehouse = (warehouseName: string, stages: Stage[]): Set<string> => {
  const reachable = new Set<string>();
  stages.forEach((stage) => {
    const stageName = stage.metadata?.name;
    if (stageName && stage.spec?.requestedFreight?.some((r) => r.origin?.name === warehouseName)) {
      reachable.add(stageName);
    }
  });
  return reachable;
};
const getStagesForImage = (
  repoURL: string,
  warehouses: Warehouse[],
  stages: Stage[]
): Set<string> => {
  const relevantWarehouses = findWarehousesForImageRepo(repoURL, warehouses);
  const allReachableStages = new Set<string>();
  relevantWarehouses.forEach((warehouseName) => {
    findStagesForWarehouse(warehouseName, stages).forEach((stageName) =>
      allReachableStages.add(stageName)
    );
  });
  return allReachableStages;
};

const ImageVersionRow = memo(
  ({
    tag,
    stagesWithImage,
    maxStageCount,
    project,
    showHistory
  }: {
    tag: string;
    stagesWithImage: Array<{ stageName: string; order: number }>;
    maxStageCount: number;
    project: string;
    showHistory: boolean;
  }) => {
    const { stageColorMap } = useContext(ColorContext);
    const navigate = useNavigate();

    return (
      <div className='bg-white rounded-md p-2.5 flex items-center gap-4 mb-2 shadow-sm'>
        <div className='w-1/5 flex-shrink-0'>
          <Tooltip title={`Image Tag: ${tag}`}>
            <span className='font-mono text-sm font-semibold text-gray-800 truncate'>{tag}</span>
          </Tooltip>
        </div>
        <div className='flex-grow flex items-center gap-2 flex-nowrap min-w-0'>
          {stagesWithImage.map(({ stageName, order }) => {
            const isHighlighted = showHistory && order === 0;

            let tooltipTitle = '';
            if (showHistory) {
              const promotionNumber = order + 1;
              tooltipTitle = isHighlighted
                ? `Most recent promotion: Currently in stage '${stageName}'`
                : `Stage: '${stageName}' (Promotion #${promotionNumber})`;
            } else {
              tooltipTitle = `Promoted to stage: '${stageName}'`;
            }

            const style = {
              backgroundColor: stageColorMap[stageName] || '#6b7280',
              opacity: showHistory && !isHighlighted ? 0.6 : 1,
              border: isHighlighted ? '2px solid white' : '1px solid transparent',
              boxShadow: isHighlighted ? '0 1px 4px rgba(0, 0, 0, 0.3)' : 'none',
              flexBasis: `calc((100% - (${maxStageCount - 1} * 0.5rem)) / ${maxStageCount})`,
              maxWidth: '40px',
              flexShrink: 0
            };

            return (
              <Tooltip key={stageName} title={tooltipTitle}>
                <div
                  onClick={() => navigate(generatePath(paths.stage, { name: project, stageName }))}
                  className='h-8 rounded-md flex items-center justify-center cursor-pointer transition-all duration-300'
                  style={style}
                >
                  {showHistory && (
                    <span className='text-white font-bold text-xs select-none'>{order + 1}</span>
                  )}
                </div>
              </Tooltip>
            );
          })}
        </div>
      </div>
    );
  }
);
ImageVersionRow.displayName = 'ImageVersionRow';

const ImageCard = memo(
  ({
    repoURL,
    tagMap,
    project,
    stages,
    warehouses,
    showHistory
  }: {
    repoURL: string;
    tagMap: TagMap;
    project: string;
    stages: Stage[];
    warehouses: Warehouse[];
    showHistory: boolean;
  }) => {
    const relevantStageNames = useMemo(
      () => getStagesForImage(repoURL, warehouses, stages),
      [repoURL, warehouses, stages]
    );

    const sortedTags = useMemo(
      () =>
        Object.keys(tagMap.tags || {}).sort((a, b) => {
          try {
            return semver.compare(b, a);
          } catch {
            return b.localeCompare(a);
          }
        }),
      [tagMap.tags]
    );

    const maxStageCount = useMemo(() => {
      const count = Math.max(
        1,
        ...sortedTags.map((tag) => {
          const imageStageMap = tagMap.tags?.[tag]?.stages || {};
          return Object.keys(imageStageMap).filter((stageName) => relevantStageNames.has(stageName))
            .length;
        })
      );
      return count;
    }, [sortedTags, tagMap.tags, relevantStageNames]);

    return (
      <div className='bg-gray-50/70 rounded-lg p-3'>
        <div className='flex flex-col'>
          {sortedTags.map((tag) => {
            const imageStageMap = tagMap.tags?.[tag]?.stages || {};
            const stagesWithImage = Object.entries(imageStageMap)
              .filter(([stageName]) => relevantStageNames.has(stageName))
              .map(([stageName, order]) => ({ stageName, order }))
              .sort((a, b) => a.order - b.order);

            if (stagesWithImage.length === 0) return null;

            return (
              <ImageVersionRow
                key={tag}
                tag={tag}
                stagesWithImage={stagesWithImage}
                maxStageCount={maxStageCount}
                project={project}
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
          'p-2 w-7 h-7 flex items-center justify-center rounded-md hover:bg-gray-300 transition-colors',
          selected ? 'bg-blue-100 text-blue-600' : 'text-gray-400 hover:text-gray-500'
        )}
      >
        <FontAwesomeIcon icon={icon} />
      </button>
    </Tooltip>
  )
);
HeaderButton.displayName = 'HeaderButton';

const Select = memo(
  ({
    value,
    onChange,
    options
  }: {
    value: string;
    onChange: (value: string) => void;
    options: { label: string; value: string }[];
  }) => (
    <select
      className='block border-gray-300 w-full text-gray-800 appearance-none p-2 bg-white rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500'
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      {options.map((option) => (
        <option value={option.value} key={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  )
);
Select.displayName = 'Select';

interface ImagesProps {
  hide: () => void;
  images: Record<string, TagMap>;
  project: string;
  stages: Stage[];
  warehouses: Warehouse[];
}

const ImagesOptimizedGrid = memo<ImagesProps>(({ hide, images, project, stages, warehouses }) => {
  const imageRepos = useMemo(() => Object.keys(images || {}), [images]);
  const [imageURL, setImageURL] = useState(imageRepos[0] || '');
  const [showHistory, setShowHistory] = useLocalStorage(`${project}-images-history`, true);

  useEffect(() => {
    if (!imageRepos.includes(imageURL) && imageRepos.length > 0) {
      setImageURL(imageRepos[0]);
    }
  }, [imageRepos, imageURL]);

  return (
    <div className='bg-white rounded-lg p-4 text-sm shadow-xl border border-gray-200/75'>
      <div className='flex items-center justify-between mb-4 px-1'>
        <h3 className='text-base font-semibold text-gray-800 flex items-center'>
          <FontAwesomeIcon icon={faDocker} className='mr-2 text-gray-500' />
          ImagesOptimizedGrid
        </h3>
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

      <div className='space-y-4'>
        {imageRepos.length > 0 && (
          <Select
            value={imageURL}
            onChange={setImageURL}
            options={imageRepos.map((repo) => ({
              label: repo.split('/').pop() || repo,
              value: repo
            }))}
          />
        )}
      </div>

      <div className='mt-4 max-h-[356px] overflow-y-auto pr-2'>
        {imageURL && images[imageURL] ? (
          <ImageCard
            repoURL={imageURL}
            tagMap={images[imageURL]}
            project={project}
            stages={stages}
            warehouses={warehouses}
            showHistory={showHistory}
          />
        ) : (
          <div className='text-center text-gray-500 py-16'>
            {imageRepos.length > 0 ? 'Select an image repository' : 'No images found'}
          </div>
        )}
      </div>
    </div>
  );
});
ImagesOptimizedGrid.displayName = 'ImagesOptimizedGrid';

export { ImagesOptimizedGrid };
