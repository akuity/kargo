import { IconDefinition, faHistory, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo, useContext, useMemo, useState, useEffect } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';
import semver from 'semver';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { TagMap, ImageStageMap } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

type ProcessedTagMap = {
  tags: Record<string, ImageStageMap>;
};

type ProcessedImages = Record<string, ProcessedTagMap>;

const findWarehousesForImageRepo = (repoURL: string, warehouses: Warehouse[]): string[] => {
  return warehouses
    .filter((w) => w.spec?.subscriptions?.some((s) => s.image?.repoURL === repoURL))
    .map((w) => w.metadata?.name || '')
    .filter(Boolean);
};

const findStagesForWarehouse = (warehouseName: string, stages: Stage[]): Set<string> => {
  const reachableStages = new Set<string>();
  stages.forEach((stage) => {
    const stageName = stage.metadata?.name;
    if (stageName && stage.spec?.requestedFreight?.some((r) => r.origin?.name === warehouseName)) {
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
    findStagesForWarehouse(warehouseName, stages).forEach((stageName) =>
      allReachableStages.add(stageName)
    );
  });
  return allReachableStages;
};

const getTooltipTitle = (
  showHistory: boolean,
  isHighlighted: boolean,
  stageName: string,
  order: number
): string => {
  if (!showHistory) {
    return `Promoted to stage: '${stageName}'`;
  }

  if (isHighlighted) {
    return `Most recent promotion: Currently in stage '${stageName}'`;
  }

  return `Stage: '${stageName}' (Sequential promotion #${order + 1})`;
};

const filterStagesForImage = (
  imageStageMap: ImageStageMap,
  stagesForImage: Set<string>
): Record<string, number> => {
  const filteredStages: Record<string, number> = {};
  Object.entries(imageStageMap.stages || {}).forEach(([stageName, order]) => {
    if (stagesForImage.has(stageName)) {
      filteredStages[stageName] = order;
    }
  });
  return filteredStages;
};

const processTagMap = (tagMap: TagMap, stagesForImage: Set<string>): ProcessedTagMap => {
  const filteredTagMap: ProcessedTagMap = { tags: {} };

  Object.entries(tagMap.tags || {}).forEach(([tag, imageStageMap]) => {
    const filteredStages = filterStagesForImage(imageStageMap, stagesForImage);
    if (Object.keys(filteredStages).length > 0) {
      filteredTagMap.tags[tag] = { ...imageStageMap, stages: filteredStages };
    }
  });

  return filteredTagMap;
};

const StageBox = memo(
  ({
    stageName,
    order,
    showHistory,
    stageColorMap,
    project
  }: {
    stageName: string;
    order: number;
    showHistory: boolean;
    stageColorMap: Record<string, string>;
    project: string;
  }) => {
    const navigate = useNavigate();
    const isHighlighted = showHistory && order === 0;
    const baseColor = stageColorMap[stageName] || '#6b7280';

    const tooltipTitle = getTooltipTitle(showHistory, isHighlighted, stageName, order);

    const handleClick = () => {
      navigate(generatePath(paths.stage, { name: project, stageName }));
    };

    const style = {
      backgroundColor: baseColor,
      opacity: showHistory && !isHighlighted ? 0.6 : 1,
      border: isHighlighted ? '2px solid rgba(255,255,255,0.5)' : '2px solid transparent',
      boxShadow: isHighlighted ? '0 1px 3px rgba(0, 0, 0, 0.2)' : 'none'
    };

    return (
      <Tooltip title={tooltipTitle}>
        <button
          onClick={handleClick}
          className='h-6 w-full rounded flex items-center justify-center cursor-pointer transition-all duration-300 border-0 p-0'
          style={style}
        >
          {showHistory && (
            <div className='flex items-center gap-0.5'>
              <span className='text-white font-bold text-[10px] select-none'>#{order + 1}</span>
            </div>
          )}
        </button>
      </Tooltip>
    );
  }
);
StageBox.displayName = 'StageBox';

const ImageTagRow = memo(
  ({
    tag,
    imageStageMap,
    dynamicStages,
    showHistory,
    project,
    stageColorMap
  }: {
    tag: string;
    imageStageMap: ImageStageMap;
    dynamicStages: string[];
    showHistory: boolean;
    project: string;
    stageColorMap: Record<string, string>;
  }) => {
    const sequentialOrderMap = useMemo(() => {
      if (!showHistory) return {};

      const stagesWithOrder = Object.entries(imageStageMap.stages || {})
        .filter(([stageName]) => dynamicStages.includes(stageName))
        .sort(([, a], [, b]) => a - b);

      const orderMap: Record<string, number> = {};
      stagesWithOrder.forEach(([stageName], index) => {
        orderMap[stageName] = index;
      });

      return orderMap;
    }, [imageStageMap.stages, dynamicStages, showHistory]);

    return (
      <tr className='hover:bg-gray-50/70'>
        <td className='sticky left-0 bg-white hover:bg-gray-50/70 px-1 py-0.5 border-b border-gray-200 z-20'>
          <Tooltip title={`Image Tag: ${tag}`}>
            <div className='font-mono text-xs font-semibold truncate'>{tag}</div>
          </Tooltip>
        </td>
        {dynamicStages.map((stageName) => {
          const originalOrder = imageStageMap.stages?.[stageName];
          const hasImage = originalOrder !== undefined;
          const sequentialOrder = sequentialOrderMap[stageName];

          return (
            <td key={stageName} className='px-1 py-1 border-b border-gray-200'>
              <div className='flex justify-center'>
                {hasImage ? (
                  <StageBox
                    stageName={stageName}
                    order={sequentialOrder}
                    showHistory={showHistory}
                    stageColorMap={stageColorMap}
                    project={project}
                  />
                ) : (
                  <div className='w-full h-6 bg-gray-100 rounded text-xs flex items-center justify-center text-gray-400'>
                    -
                  </div>
                )}
              </div>
            </td>
          );
        })}
      </tr>
    );
  }
);
ImageTagRow.displayName = 'ImageTagRow';

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
          'p-2 w-7 h-7 flex items-center justify-center rounded-md hover:bg-gray-200 transition-colors',
          selected ? 'bg-blue-100 text-blue-600' : 'text-gray-500 hover:text-gray-700'
        )}
      >
        <FontAwesomeIcon icon={icon} />
      </button>
    </Tooltip>
  )
);
HeaderButton.displayName = 'HeaderButton';

interface ImagesProps {
  hide: () => void;
  images: Record<string, TagMap>;
  project: string;
  stages: Stage[];
  warehouses: Warehouse[];
}

const sortTags = (tags: string[]): string[] => {
  return tags.sort((a, b) => {
    try {
      return semver.compare(b, a);
    } catch {
      return b.localeCompare(a);
    }
  });
};

const getDynamicStages = (selectedImageData: ProcessedTagMap | undefined): string[] => {
  if (!selectedImageData) return [];

  const stageSet = new Set<string>();
  Object.values(selectedImageData.tags).forEach((imageStageMap) => {
    Object.keys(imageStageMap.stages || {}).forEach((stageName) => stageSet.add(stageName));
  });

  return Array.from(stageSet).sort();
};

const TableHeader = ({
  dynamicStages,
  stageColorMap
}: {
  dynamicStages: string[];
  stageColorMap: Record<string, string>;
}) => (
  <thead>
    <tr className='bg-gray-50'>
      <th className='sticky left-0 bg-gray-50 text-left px-1 py-0.5 border-b font-semibold text-gray-600 z-20'>
        Tag
      </th>
      {dynamicStages.map((stageName) => (
        <th key={stageName} className='px-1 py-1 border-b font-semibold text-gray-600 text-center'>
          <Tooltip title={stageName}>
            <div className='flex items-center justify-center gap-1'>
              <div
                className='w-2.5 h-2.5 rounded-full flex-shrink-0'
                style={{ backgroundColor: stageColorMap[stageName] }}
              />
              <span className='text-xs font-medium text-gray-700 truncate max-w-[60px]'>
                {stageName}
              </span>
            </div>
          </Tooltip>
        </th>
      ))}
    </tr>
  </thead>
);

const TableBody = ({
  allTags,
  selectedImageData,
  dynamicStages,
  showHistory,
  project,
  stageColorMap,
  repoURLs
}: {
  allTags: string[];
  selectedImageData: ProcessedTagMap | undefined;
  dynamicStages: string[];
  showHistory: boolean;
  project: string;
  stageColorMap: Record<string, string>;
  repoURLs: string[];
}) => (
  <tbody>
    {allTags.length > 0 ? (
      allTags.map((tag) => {
        const imageStageMap = selectedImageData?.tags[tag];
        if (!imageStageMap) return null;
        return (
          <ImageTagRow
            key={tag}
            tag={tag}
            imageStageMap={imageStageMap}
            dynamicStages={dynamicStages}
            showHistory={showHistory}
            project={project}
            stageColorMap={stageColorMap}
          />
        );
      })
    ) : (
      <tr>
        <td colSpan={dynamicStages.length + 1} className='text-center text-gray-500 py-12'>
          {repoURLs.length > 0 ? 'Select an image' : 'No images found'}
        </td>
      </tr>
    )}
  </tbody>
);

export const Images = memo<ImagesProps>(({ hide, images, project, stages, warehouses }) => {
  const { stageColorMap } = useContext(ColorContext);
  const [showHistory, setShowHistory] = useLocalStorage('images-dynamic-grid-show-history', true);

  const filteredImages: ProcessedImages = useMemo(() => {
    const result: ProcessedImages = {};

    Object.entries(images).forEach(([repoURL, tagMap]) => {
      const stagesForImage = getStagesForImage(repoURL, warehouses, stages);
      const processedTagMap = processTagMap(tagMap, stagesForImage);

      if (Object.keys(processedTagMap.tags).length > 0) {
        result[repoURL] = processedTagMap;
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

  const selectedImageData = useMemo(
    () => (selectedRepoURL ? filteredImages[selectedRepoURL] : undefined),
    [selectedRepoURL, filteredImages]
  );

  const dynamicStages = useMemo(() => getDynamicStages(selectedImageData), [selectedImageData]);

  const allTags = useMemo(() => {
    if (!selectedImageData) return [];
    const tags = Object.keys(selectedImageData.tags);
    return sortTags(tags);
  }, [selectedImageData]);

  return (
    <div className='bg-white rounded-lg shadow-xl border border-gray-200/75 p-3'>
      <div className='flex items-center justify-between mb-2'>
        <div className='flex items-baseline gap-2'>
          <h3 className='text-base font-semibold text-gray-800'>Images</h3>
          {selectedRepoURL && (
            <p className='text-sm text-gray-600 truncate'>{selectedRepoURL.split('/').pop()}</p>
          )}
        </div>
        <div className='flex items-center gap-1'>
          <HeaderButton
            onClick={() => setShowHistory(!showHistory)}
            selected={showHistory}
            icon={faHistory}
            title={showHistory ? 'Hide promotion history' : 'Show promotion history'}
          />
          <HeaderButton onClick={hide} icon={faEyeSlash} title='Hide panel' />
        </div>
      </div>
      {showHistory && (
        <div className='mb-1 text-xs text-gray-500 bg-gray-50 px-2 py-0.5 rounded'>
          Numbers show sequential promotion order (#1 = most recent)
        </div>
      )}

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

      <div className='overflow-x-auto max-h-[356px] relative z-10'>
        <table className='text-sm border-collapse w-full' style={{ tableLayout: 'fixed' }}>
          <colgroup>
            <col style={{ width: '120px' }} />
            {dynamicStages.map((stageName) => (
              <col key={stageName} style={{ width: '80px' }} />
            ))}
          </colgroup>
          <TableHeader dynamicStages={dynamicStages} stageColorMap={stageColorMap} />
          <TableBody
            allTags={allTags}
            selectedImageData={selectedImageData}
            dynamicStages={dynamicStages}
            showHistory={showHistory}
            project={project}
            stageColorMap={stageColorMap}
            repoURLs={repoURLs}
          />
        </table>
      </div>
    </div>
  );
});

Images.displayName = 'Images';
