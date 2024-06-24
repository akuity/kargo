import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Switch, Tooltip } from 'antd';
import classNames from 'classnames';
import { useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { StagePixelStyle, StageStyleMap } from './types';
import { useImages } from './utils/useImages';

const ImageTagRow = ({
  tag,
  stages,
  stylesByStage,
  projectName,
  showHistory
}: {
  tag: string;
  stages: Stage[];
  stylesByStage: StageStyleMap;
  projectName: string;
  showHistory: boolean;
}) => {
  const navigate = useNavigate();
  return (
    <div className='flex items-center mb-2'>
      <Tooltip title={tag}>
        <div className='mr-4 font-mono text-sm text-right w-20 truncate'>{tag}</div>
      </Tooltip>
      {stages.map((stage) => {
        let curStyles: StagePixelStyle | null = stylesByStage[stage.metadata?.name || ''];
        if (curStyles) {
          if (!showHistory && curStyles.opacity < 1) {
            curStyles = null;
          } else if (showHistory && curStyles.opacity == 1) {
            curStyles = {
              ...curStyles,
              border: '3px solid rgba(255,255,255,0.3)'
            };
          }
        }
        return (
          <Tooltip key={stage.metadata?.name} title={stage.metadata?.name}>
            <div
              className={classNames('mr-2 bg-neutral-300 ', {
                'cursor-pointer': !!curStyles
              })}
              style={{
                borderRadius: '5px',
                border: '3px solid transparent',
                height: '30px',
                width: '30px',
                ...curStyles
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

export const Images = (props: { project: string; stages: Stage[] }) => {
  return (
    <div
      className='text-neutral-600 text-sm bg-neutral-100'
      style={{
        width: '400px'
      }}
    >
      <h3 className='bg-neutral-200 px-4 py-2 flex items-center text-sm text-neutral-500'>
        <FontAwesomeIcon icon={faDocker} className='mr-2' /> IMAGES
      </h3>
      <div className='p-4'>
        <ImagesTable {...props} />
      </div>
    </div>
  );
};

export const ImagesTable = ({ project, stages }: { project: string; stages: Stage[] }) => {
  const images = useImages(stages);

  const [imageURL, setImageURL] = useState(images.keys().next().value as string);
  const [showHistory, setShowHistory] = useLocalStorage(`${project}-show-history`, false);

  useEffect(() => {
    setImageURL(images.keys().next().value as string);
  }, [images]);

  const curImage = useMemo(() => {
    return images.get(imageURL);
  }, [imageURL]);

  return (
    <>
      {curImage ? (
        <>
          <div className='mb-4 flex items-center'>
            <Switch onChange={(val) => setShowHistory(val)} checked={showHistory} />
            <div className='ml-2 font-semibold'>SHOW HISTORY</div>
          </div>
          <div className='mb-8'>
            <Select
              value={imageURL}
              onChange={(value) => setImageURL(value as string)}
              options={Array.from(images.keys()).map((image) => ({
                label: image.split('/').pop(),
                value: image
              }))}
            />
          </div>
          {Array.from(curImage.entries())
            .sort((a, b) => b[0].localeCompare(a[0], undefined, { numeric: true }))
            .map(([tag, tagStages]) => (
              <ImageTagRow
                key={tag}
                projectName={project}
                tag={tag}
                stages={stages}
                stylesByStage={tagStages}
                showHistory={showHistory}
              />
            ))}
        </>
      ) : (
        <p>No images available</p>
      )}
    </>
  );
};

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
    className='block border-none w-full text-neutral-600 appearance-none p-2 bg-neutral-200 focus:outline-none focus:ring-2 focus:ring-blue-400'
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
