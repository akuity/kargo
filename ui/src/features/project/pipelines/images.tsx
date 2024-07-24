import { faDocker } from '@fortawesome/free-brands-svg-icons';
import { IconDefinition, faBook, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { useContext, useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';
import { useLocalStorage } from '@ui/utils/use-local-storage';

interface StagePixelStyle {
  opacity: number;
  backgroundColor: string;
  border?: string;
}

type StageStyleMap = { [key: string]: StagePixelStyle };

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
              className={classNames('mr-2 bg-gray-300 ', {
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

export const Images = ({
  project,
  stages,
  hide
}: {
  project: string;
  stages: Stage[];
  hide: () => void;
}) => {
  const { stageColorMap: colors } = useContext(ColorContext);
  const images = useMemo(() => {
    const images = new Map<string, Map<string, StageStyleMap>>();
    stages.forEach((stage) => {
      const len = stage.status?.freightHistory?.length || 0;
      stage.status?.freightHistory?.forEach((freightGroup, i) => {
        (Object.values(freightGroup.items || {}) || []).forEach((freight) => {
          freight.images?.forEach((image) => {
            let repo = image.repoURL ? images.get(image.repoURL) : undefined;
            if (!repo) {
              repo = new Map<string, StageStyleMap>();
              images.set(image.repoURL!, repo);
            }
            let curStages = image.tag ? repo.get(image.tag) : undefined;
            if (!curStages) {
              curStages = {} as StageStyleMap;
            }
            curStages[stage.metadata?.name as string] = {
              opacity: 1 - i / len,
              backgroundColor: colors[stage.metadata?.name as string]
            };
            repo.set(image.tag!, curStages);
          });
        });
      });

      stage.status?.currentFreight?.images?.forEach((image) => {
        let repo = image.repoURL ? images.get(image.repoURL) : undefined;
        if (!repo) {
          repo = new Map<string, StageStyleMap>();
          images.set(image.repoURL!, repo);
        }
        let curStages = image.tag ? repo.get(image.tag) : undefined;
        if (!curStages) {
          curStages = {} as StageStyleMap;
        }
        curStages[stage.metadata?.name as string] = {
          opacity: 1,
          backgroundColor: colors[stage.metadata?.name as string]
        };
        repo.set(image.tag!, curStages);
      });
    });
    return images;
  }, [stages]);

  const [imageURL, setImageURL] = useState(images.keys().next().value as string);
  const [showHistory, setShowHistory] = useLocalStorage(`${project}-show-history`, true);

  useEffect(() => {
    setImageURL(images.keys().next().value as string);
  }, [images]);

  const curImage = useMemo(() => {
    return images.get(imageURL);
  }, [imageURL]);

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
      </div>
    </div>
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
