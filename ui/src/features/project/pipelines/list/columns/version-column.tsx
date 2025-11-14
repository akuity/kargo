import { Flex, Typography } from 'antd';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getCurrentFreight } from '@ui/features/common/utils';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { FreightArtifact } from '@ui/features/project/pipelines/freight/freight-artifact';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

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
            <FreightArtifact expand key={commit?.repoURL} artifact={commit} />
          ))}

          {firstFreight?.charts?.slice(0, 2).map((chart) => (
            <FreightArtifact expand key={chart?.repoURL} artifact={chart} />
          ))}

          {firstFreight?.images?.slice(0, 2).map((image) => (
            <FreightArtifact expand key={image?.repoURL} artifact={image} />
          ))}

          {totalArtifacts > 6 && (
            <Typography.Text type='secondary' className='text-[10px]'>
              +{' '}
              {totalArtifacts -
                (firstFreight?.charts?.slice(0, 2)?.length +
                  firstFreight?.commits?.slice(0, 2)?.length +
                  firstFreight?.images?.slice(0, 2)?.length)}{' '}
              more
            </Typography.Text>
          )}
        </div>
      </Flex>
    );
  }
});
