import { Select, Tooltip } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getStageColors } from '@ui/features/stage/utils';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

export const Images = ({ projectName, stages }: { projectName: string; stages: Stage[] }) => {
  const [images, colors] = useMemo(() => {
    const images = new Map<string, Map<string, Set<string>>>();
    stages.forEach((stage) => {
      stage.status?.currentFreight?.images?.forEach((image) => {
        let repo = images.get(image.repoUrl);
        if (!repo) {
          repo = new Map<string, Set<string>>();
          images.set(image.repoUrl, repo);
        }
        let stages = repo.get(image.tag);
        if (!stages) {
          stages = new Set<string>();
          repo.set(image.tag, stages);
        }
        stages.add(stage.metadata?.name as string);
      });
    });
    return [images, getStageColors([...stages])];
  }, [stages]);

  const navigate = useNavigate();
  const [imageURL, setImageURL] = useState(images.keys().next().value as string);
  const image = imageURL && images.get(imageURL);

  return (
    <>
      {image ? (
        <>
          <div className='mb-8'>
            <Select
              className='w-full'
              value={imageURL}
              onChange={(value) => setImageURL(value as string)}
              options={Array.from(images.keys()).map((image) => ({
                label: image.split('/').pop(),
                value: image
              }))}
            />
          </div>
          {Array.from(image.entries())
            .sort((a, b) => b[0].localeCompare(a[0], undefined, { numeric: true }))
            .map(([tag, tagStages]) => (
              <div key={tag} className='flex items-center mb-2'>
                <Tooltip title={tag}>
                  <div className='mr-4 font-mono text-sm text-right w-20 truncate'>{tag}</div>
                </Tooltip>
                {stages.map((stage) => (
                  <Tooltip key={stage.metadata?.name} title={stage.metadata?.name}>
                    <div
                      className={classNames('mr-2 bg-zinc-600', {
                        'cursor-pointer': tagStages.has(stage.metadata?.name || '')
                      })}
                      style={{
                        borderRadius: '5px',
                        height: '30px',
                        width: '30px',
                        backgroundColor: tagStages.has(stage.metadata?.name || '')
                          ? colors[stage.metadata?.uid || '']
                          : ''
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
                ))}
              </div>
            ))}
        </>
      ) : (
        <p>No images available</p>
      )}
    </>
  );
};
