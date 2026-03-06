import { faHistory, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { Card, Flex, Select, Table, Tooltip, Typography } from 'antd';
import { ColumnType } from 'antd/es/table';
import { memo, useContext, useMemo, useState, useEffect } from 'react';

import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { TagMap } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { shortVersion } from '../freight/short-version-utils';

import { getStages, getStagesForImage } from './get-stages';
import { HeaderButton } from './header-button';
import { processTagMap } from './process-tag-map';
import { sortTags } from './sort-tags';
import { StageBox } from './stage-box';
import { ProcessedImages } from './types';
import { usePromotionHistory } from './use-promotion-history';

interface ImagesProps {
  hide: () => void;
  images: Record<string, TagMap>;
  project: string;
  stages: Stage[];
  warehouses: WarehouseExpanded[];
}

export const Images = memo<ImagesProps>(({ hide, images, project, stages, warehouses }) => {
  const { stageColorMap } = useContext(ColorContext);
  const [showHistory, setShowHistory] = useLocalStorage('images-dynamic-grid-show-history', true);

  const filteredImages: ProcessedImages = useMemo(() => {
    const result: ProcessedImages = {};

    for (const [repoURL, tagMap] of Object.entries(images)) {
      const stagesForImage = getStagesForImage(repoURL, warehouses, stages);
      const processedTagMap = processTagMap(tagMap, stagesForImage);

      if (Object.keys(processedTagMap.tags).length > 0) {
        result[repoURL] = processedTagMap;
      }
    }

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

  const promotionHistory = usePromotionHistory(stages);

  const selectedImageData = useMemo(
    () => (selectedRepoURL ? filteredImages[selectedRepoURL] : undefined),
    [selectedRepoURL, filteredImages]
  );

  const dynamicStages = useMemo(() => getStages(selectedImageData), [selectedImageData]);

  const allTags = useMemo(() => {
    if (!selectedImageData) return [];
    const tags = Object.keys(selectedImageData.tags);
    return sortTags(tags);
  }, [selectedImageData]);

  const tableColumns = useMemo(() => {
    const columns: ColumnType<{ tag: string }>[] = [
      {
        title: 'Tag',
        key: 'tag',
        width: 120,
        fixed: 'left',
        render: (_: unknown, record: { tag: string }) => (
          <Tooltip title={`Image Tag: ${record.tag}`}>
            <Typography.Text className='font-mono text-xs font-semibold truncate'>
              {shortVersion(record.tag, 15)}
            </Typography.Text>
          </Tooltip>
        )
      }
    ];

    for (const stageName of dynamicStages) {
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

          // Get promotion orders from promotion history
          const tagHistory = promotionHistory[selectedRepoURL]?.[record.tag]?.[stageName] || [];
          const sortedOrders =
            tagHistory.length > 0 ? tagHistory : [imageStageMap.stages?.[stageName] || 1];

          return (
            <div className='flex justify-center'>
              <StageBox
                stageName={stageName}
                orders={sortedOrders}
                showHistory={showHistory}
                stageColorMap={stageColorMap}
                project={project}
              />
            </div>
          );
        }
      });
    }

    return columns;
  }, [
    dynamicStages,
    stageColorMap,
    selectedImageData,
    showHistory,
    project,
    promotionHistory,
    selectedRepoURL
  ]);

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
          Numbers show promotion order (1 = most recent)
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
