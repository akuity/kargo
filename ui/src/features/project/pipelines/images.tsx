import { IconDefinition, faHistory, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Select, Table, Tooltip, Typography } from 'antd';
import { ColumnType } from 'antd/es/table';
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
        <Button
          onClick={handleClick}
          className='h-6 w-full rounded flex items-center justify-center cursor-pointer transition-all duration-300 border-0 p-0'
          style={style}
        >
          {showHistory && (
            <div className='flex items-center gap-0.5'>
              <span className='text-white font-bold text-[10px] select-none'>#{order + 1}</span>
            </div>
          )}
        </Button>
      </Tooltip>
    );
  }
);
StageBox.displayName = 'StageBox';

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
      <Button
        onClick={onClick}
        className={classNames(
          'p-2 w-7 h-7 flex items-center justify-center rounded-md hover:bg-gray-200 transition-colors',
          selected ? 'bg-blue-100 text-blue-600' : 'text-gray-500 hover:text-gray-700'
        )}
      >
        <FontAwesomeIcon icon={icon} />
      </Button>
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

  const sequentialOrderMaps = useMemo(() => {
    if (!selectedImageData || !showHistory) return {};

    const maps: Record<string, Record<string, number>> = {};

    Object.entries(selectedImageData.tags).forEach(([tag, imageStageMap]) => {
      const stagesWithOrder = Object.entries(imageStageMap.stages || {})
        .filter(([stageName]) => dynamicStages.includes(stageName))
        .sort(([, a], [, b]) => a - b);

      const orderMap: Record<string, number> = {};
      stagesWithOrder.forEach(([stageName], index) => {
        orderMap[stageName] = index;
      });

      maps[tag] = orderMap;
    });

    return maps;
  }, [selectedImageData, dynamicStages, showHistory]);

  const tableColumns = useMemo(() => {
    const columns: ColumnType<{ tag: string }>[] = [
      {
        title: 'Tag',
        key: 'tag',
        width: 120,
        render: (_: unknown, record: { tag: string }) => (
          <Tooltip title={`Image Tag: ${record.tag}`}>
            <Typography.Text className='font-mono text-xs font-semibold truncate'>
              {record.tag}
            </Typography.Text>
          </Tooltip>
        )
      }
    ];

    dynamicStages.forEach((stageName) => {
      columns.push({
        title: (
          <Tooltip title={stageName}>
            <Flex align='center' justify='center' gap={4}>
              <div
                className='w-2.5 h-2.5 rounded-full flex-shrink-0'
                style={{ backgroundColor: stageColorMap[stageName] }}
              />
              <Typography.Text className='text-xs font-medium text-gray-700 truncate max-w-[60px]'>
                {stageName}
              </Typography.Text>
            </Flex>
          </Tooltip>
        ),
        key: stageName,
        width: 80,
        render: (_: unknown, record: { tag: string }) => {
          const imageStageMap = selectedImageData?.tags[record.tag];
          if (!imageStageMap) {
            return (
              <div className='w-full h-6 bg-gray-100 rounded text-xs flex items-center justify-center text-gray-400'>
                -
              </div>
            );
          }

          const originalOrder = imageStageMap.stages?.[stageName];
          const hasImage = originalOrder !== undefined;

          if (!hasImage) {
            return (
              <div className='w-full h-6 bg-gray-100 rounded text-xs flex items-center justify-center text-gray-400'>
                -
              </div>
            );
          }

          const sequentialOrder = sequentialOrderMaps[record.tag]?.[stageName] ?? 0;

          return (
            <div className='flex justify-center'>
              <StageBox
                stageName={stageName}
                order={sequentialOrder}
                showHistory={showHistory}
                stageColorMap={stageColorMap}
                project={project}
              />
            </div>
          );
        }
      });
    });

    return columns;
  }, [dynamicStages, stageColorMap, selectedImageData, showHistory, project, sequentialOrderMaps]);

  const tableData = useMemo(() => {
    return allTags.map((tag) => ({
      key: tag,
      tag
    }));
  }, [allTags]);

  return (
    <Card className='shadow-xl border border-gray-200/75'>
      <Flex justify='space-between' align='center' className='mb-2'>
        <Flex align='baseline' gap={8}>
          <Typography.Title level={4} className='!mb-0'>
            Images
          </Typography.Title>
          {selectedRepoURL && (
            <Typography.Text type='secondary' className='truncate'>
              {selectedRepoURL.split('/').pop()}
            </Typography.Text>
          )}
        </Flex>
        <Flex gap={4}>
          <HeaderButton
            onClick={() => setShowHistory(!showHistory)}
            selected={showHistory}
            icon={faHistory}
            title={showHistory ? 'Hide promotion history' : 'Show promotion history'}
          />
          <HeaderButton onClick={hide} icon={faEyeSlash} title='Hide panel' />
        </Flex>
      </Flex>

      {showHistory && (
        <Typography.Text
          type='secondary'
          className='text-xs bg-gray-50 px-2 py-0.5 rounded block mb-2'
        >
          Numbers show sequential promotion order (#1 = most recent)
        </Typography.Text>
      )}

      {repoURLs.length > 1 && (
        <div className='mb-3'>
          <Select
            value={selectedRepoURL}
            onChange={setSelectedRepoURL}
            className='w-full'
            options={repoURLs.map((repo) => ({
              value: repo,
              label: repo.split('/').pop() || repo
            }))}
          />
        </div>
      )}

      <Table
        columns={tableColumns}
        dataSource={tableData}
        pagination={false}
        size='small'
        scroll={{ x: 'max-content', y: 356 }}
        locale={{
          emptyText: (
            <Typography.Text type='secondary' className='py-12'>
              {repoURLs.length > 0 ? 'Select an image' : 'No images found'}
            </Typography.Text>
          )
        }}
      />
    </Card>
  );
});

Images.displayName = 'Images';
