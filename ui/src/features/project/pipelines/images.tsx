import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faBook, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo, useContext, useEffect, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';
import semver from 'semver';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { TagMap } from '@ui/gen/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

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
        const cur = imageStageMap?.[stage.metadata?.name || ''];
        const len = stages?.length || 0;

        let opacity = 0;

        if (!isNaN(cur)) {
          opacity = len > 0 && showHistory ? 1 - cur / len : 1;
        }

        return (
          <Tooltip key={stage.metadata?.name} title={stage.metadata?.name}>
            <div
              className={classNames('mr-2 bg-gray-300 ', {
                'cursor-pointer': cur >= 0
              })}
              style={{
                borderRadius: '5px',
                height: '30px',
                width: '30px',
                opacity,
                backgroundColor:
                  (showHistory && cur >= 0) || cur === 0
                    ? stageColorMap?.[stage?.metadata?.name || '']
                    : undefined,
                border:
                  cur === 0 && showHistory
                    ? '3px solid rgba(255,255,255,0.3)'
                    : '3px solid transparent'
              }}
              onClick={() =>
                navigate(
                  generatePath(paths.stage, {
                    name: projectName,
                    stageName: stage.metadata?.name
                  })
                )
              }
            />
          </Tooltip>
        );
      })}
    </div>
  );
};

const HeaderButton = ({
  onClick,
  selected,
  className,
  icon
}: {
  onClick: () => void;
  icon: IconDefinition;
  selected?: boolean;
  className?: string;
}) => (
  <div
    onClick={onClick}
    className={classNames(
      'cursor-pointer',
      {
        'text-blue-500': selected
      },
      className
    )}
  >
    <FontAwesomeIcon icon={icon} />
  </div>
);

// IMPORTANT: keep this wrapped in memo please
export const Images = memo(
  ({
    project,
    stages,
    hide,
    images
  }: {
    project: string;
    stages: Stage[];
    hide: () => void;
    images: { [key: string]: TagMap };
  }) => {
    const [imageURL, setImageURL] = useState(Object.keys(images || {})?.[0]);
    const [showHistory, setShowHistory] = useLocalStorage(`${project}-show-history`, true);

    useEffect(() => {
      setImageURL(Object.keys(images || {})?.[0]);
    }, [images]);

    const curImage = images[imageURL];

    const sortedTags = curImage?.tags
      ? (Object.keys(curImage.tags || {}) || []).sort((a, b) => {
          try {
            return semver.compare(b, a);
          } catch (e) {
            // no chance of dirty semver but just-in-case
            return 0;
          }
        })
      : [];

    return (
      <div className='text-gray-600 text-sm bg-gray-100 pb-4 rounded-md overflow-hidden'>
        <h3 className='bg-gray-200 px-4 py-2 flex items-center text-sm text-gray-500'>
          <FontAwesomeIcon icon={faDocker} className='mr-2' /> IMAGES
          <Tooltip title='Show history'>
            <div className='ml-auto'>
              <HeaderButton
                onClick={() => setShowHistory(!showHistory)}
                icon={faBook}
                selected={showHistory}
              />
            </div>
          </Tooltip>
          <HeaderButton onClick={hide} icon={faEyeSlash} className='ml-2' />
        </h3>
        <div className='p-4'>
          {curImage ? (
            <>
              <div className='mb-8'>
                <Select
                  value={imageURL}
                  onChange={(value) => setImageURL(value as string)}
                  options={Object.keys(images || []).map((image) => ({
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
                  stages={stages}
                  imageStageMap={curImage.tags[tag]?.stages}
                  showHistory={showHistory}
                />
              ))}
            </>
          ) : (
            <p>No images available</p>
          )}
        </div>
      </div>
    );
  }
);

const Select = ({
  value,
  onChange,
  options
}: {
  value: string;
  onChange: (value: string) => void;
  options: { label?: string; value: string }[];
}) => (
  <select
    className='block border-none w-full text-gray-600 appearance-none p-2 bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-400'
    value={value}
    onChange={(e) => onChange(e.target.value)}
  >
    {options.map((option) => (
      <option value={option.value} key={option.label}>
        {option.label}
      </option>
    ))}
  </select>
);
