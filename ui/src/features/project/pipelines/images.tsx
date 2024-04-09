import { Switch, Tooltip } from 'antd';
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
              className={classNames('mr-2 bg-zinc-600 ', {
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

export const Images = ({ projectName, stages }: { projectName: string; stages: Stage[] }) => {
  const colors = useContext(ColorContext);
  const images = useMemo(() => {
    const images = new Map<string, Map<string, StageStyleMap>>();
    stages.forEach((stage) => {
      const len = stage.status?.history?.length || 0;
      stage.status?.history?.forEach((freight, i) => {
        freight.images?.forEach((image) => {
          let repo = image.repoURL ? images.get(image.repoURL) : undefined;
          if (!repo) {
            repo = new Map<string, StageStyleMap>();
            images.set(image.repoURL!, repo);
          }
          let stages = image.tag ? repo.get(image.tag) : undefined;
          if (!stages) {
            stages = {} as StageStyleMap;
            repo.set(image.tag!, stages);
          }
          stages[stage.metadata?.name as string] = {
            opacity: 1 - i / len,
            backgroundColor: colors[stage.metadata?.name as string]
          };
        });
      });

      stage.status?.currentFreight?.images?.forEach((image) => {
        let repo = image.repoURL ? images.get(image.repoURL) : undefined;
        if (!repo) {
          repo = new Map<string, StageStyleMap>();
          images.set(image.repoURL!, repo);
        }
        let stages = image.tag ? repo.get(image.tag) : undefined;
        if (!stages) {
          stages = {} as StageStyleMap;
          repo.set(image.tag!, stages);
        }
        stages[stage.metadata?.name as string] = {
          opacity: 1,
          backgroundColor: colors[stage.metadata?.name as string]
        };
      });
    });
    return images;
  }, [stages]);

  const [imageURL, setImageURL] = useState(images.keys().next().value as string);
  const [curImage, setCurImage] = useState(imageURL && images.get(imageURL));
  const [showHistory, setShowHistory] = useLocalStorage(`${projectName}-show-history`, false);

  useEffect(() => {
    setImageURL(images.keys().next().value as string);
  }, [images]);

  useEffect(() => {
    setCurImage(images.get(imageURL));
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
                projectName={projectName}
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
    className='block border-none w-full text-gray appearance-none p-2 bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-blue-400'
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
