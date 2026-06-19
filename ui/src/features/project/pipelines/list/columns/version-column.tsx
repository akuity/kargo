import { Flex, Typography } from 'antd';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getCurrentFreight } from '@ui/features/common/utils';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import {
  ChartArtifact,
  GitCommitArtifact,
  ImageArtifact
} from '@ui/features/project/pipelines/freight/freight-artifact';
import { Stage } from '@ui/gen/api/v2/models';

export const versionColumn = (): ColumnType<Stage> => ({
  title: 'Version',
  render: (_, stage) => {
    const dictionaryContext = useDictionaryContext();
    const freightTimelineControllerContext = useFreightTimelineControllerContext();
    const currentFreight = getCurrentFreight(stage);

    // TODO: filter by sources
    const firstFreight = currentFreight[0];

    const totalArtifacts =
      (firstFreight?.commits?.length || 0) +
      (firstFreight?.charts?.length || 0) +
      (firstFreight?.images?.length || 0);

    const alias = dictionaryContext?.freightById?.[firstFreight?.name || '']?.alias;

    return (
      <Flex vertical gap={8}>
        {alias && freightTimelineControllerContext?.preferredFilter?.showAlias ? (
          <Link
            className='text-xs'
            to={generatePath(paths.freight, {
              name: stage?.metadata?.namespace,
              freightName: firstFreight?.name
            })}
          >
            {alias}
          </Link>
        ) : null}

        <div>
          {firstFreight?.commits?.slice(0, 2).map((commit) => (
            <GitCommitArtifact expand key={commit?.repoURL} commit={commit} />
          ))}

          {firstFreight?.charts?.slice(0, 2).map((chart) => (
            <ChartArtifact expand key={chart?.repoURL} chart={chart} />
          ))}

          {firstFreight?.images?.slice(0, 2).map((image) => (
            <ImageArtifact expand key={image?.repoURL} image={image} />
          ))}

          {totalArtifacts > 6 && (
            <Typography.Text type='secondary' className='text-[10px]'>
              +{' '}
              {totalArtifacts -
                ((firstFreight?.charts?.slice(0, 2)?.length || 0) +
                  (firstFreight?.commits?.slice(0, 2)?.length || 0) +
                  (firstFreight?.images?.slice(0, 2)?.length || 0))}{' '}
              more
            </Typography.Text>
          )}
        </div>
      </Flex>
    );
  }
});
