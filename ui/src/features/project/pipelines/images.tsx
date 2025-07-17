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

const ImageTagRow = ({
  tag,
  stages,
  imageStageMap,
  projectName,
  showHistory
}: {
  tag: string;
  stages: Stage[];
  imageStageMap: { [key: string]: number };
  projectName: string;
  showHistory: boolean;
}) => {
  const { stageColorMap } = useContext(ColorContext);
  const navigate = useNavigate();

  return (
    <div className='flex items-center mb-2'>
      <Tooltip title={tag}>
        <div className='mr-4 font-mono text-sm text-right w-20 truncate'>{tag}</div>
      </Tooltip>
      {stages.map((stage) => {
        const stageName = stage.metadata?.name || '';
        const cur = imageStageMap?.[stageName];
        const len = stages.length;

        let opacity = 0;
        if (cur !== undefined && !isNaN(cur)) {
          opacity = len > 0 && showHistory ? 1 - cur / len : 1;
        }

        return (
          <Tooltip key={stageName} title={stageName}>
            <div
              className={classNames('mr-2 transition-all duration-200', {
                'cursor-pointer': cur !== undefined && cur >= 0
              })}
              style={{
                borderRadius: '5px',
                height: '30px',
                width: '30px',
                opacity,
                backgroundColor:
                  (showHistory && cur !== undefined && cur >= 0) || cur === 0
                    ? stageColorMap?.[stageName]
                    : '#e5e7eb',
                border:
                  cur === 0 && showHistory
                    ? '3px solid rgba(255, 255, 255, 0.4)'
                    : '3px solid transparent'
              }}
              onClick={() => {
                if (cur !== undefined) {
                  navigate(
                    generatePath(paths.stage, {
                      name: projectName,
                      stageName: stageName
                    })
                  );
                }
              }}
            />
          </Tooltip>
        );
      })}
    </div>
  );
};

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
    options: { label?: string; value: string }[];
  }) => (
    <select
      className='block border-none w-full text-gray-600 appearance-none p-2 bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-400 rounded-md'
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

export const Images = memo(
  ({
    project,
    stages,
    hide,
    images,
    warehouses
  }: {
    project: string;
    stages: Stage[];
    hide: () => void;
    images: { [key: string]: TagMap };
    warehouses: Warehouse[];
  }) => {
    const imageRepos = useMemo(() => Object.keys(images || {}), [images]);
    const [imageURL, setImageURL] = useState(imageRepos[0] || '');
    const [showHistory, setShowHistory] = useLocalStorage(`${project}-show-history`, true);

    useEffect(() => {
      if (!imageRepos.includes(imageURL)) {
        setImageURL(imageRepos[0] || '');
      }
    }, [imageRepos, imageURL]);

    const relevantStages = useMemo(
      () => (imageURL ? getStagesForImage(imageURL, warehouses, stages) : new Set<string>()),
      [imageURL, warehouses, stages]
    );

    const filteredStages = useMemo(
      () => stages.filter((stage) => relevantStages.has(stage.metadata?.name || '')),
      [stages, relevantStages]
    );

    const curImage = images[imageURL];
    const sortedTags = useMemo(() => {
      return curImage?.tags
        ? Object.keys(curImage.tags || {}).sort((a, b) => {
            try {
              return semver.compare(b, a);
            } catch (e) {
              return b.localeCompare(a);
            }
          })
        : [];
    }, [curImage]);

    return (
      <div className='text-gray-600 text-sm bg-gray-100 pb-4 rounded-md'>
        <h3 className='bg-gray-200 px-4 py-2 flex items-center text-sm text-gray-500'>
          <FontAwesomeIcon icon={faDocker} className='mr-2' /> IMAGES
          <div className='ml-auto flex items-center gap-2'>
            <HeaderButton
              onClick={() => setShowHistory(!showHistory)}
              icon={faBook}
              selected={showHistory}
              title={showHistory ? 'Hide promotion history' : 'Show promotion history'}
            />
            <HeaderButton onClick={hide} icon={faEyeSlash} title='Hide images panel' />
          </div>
        </h3>
        <div className='p-4 overflow-y-auto max-h-[356px]'>
          {imageRepos.length > 0 && imageURL ? (
            <>
              <div className='mb-4'>
                <Select
                  value={imageURL}
                  onChange={setImageURL}
                  options={imageRepos.map((image) => ({
                    label: image.split('/').pop(),
                    value: image
                  }))}
                />
              </div>
              {sortedTags.map((tag) => (
                <ImageTagRow
                  key={tag}
                  projectName={project}
                  tag={tag}
                  stages={filteredStages}
                  imageStageMap={curImage.tags[tag]?.stages || {}}
                  showHistory={showHistory}
                />
              ))}
            </>
          ) : (
            <p className='text-center text-gray-500 py-8'>No images available for this project.</p>
          )}
        </div>
      </div>
    );
  }
);
Images.displayName = 'Images';
