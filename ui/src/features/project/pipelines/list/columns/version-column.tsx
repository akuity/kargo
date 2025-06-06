import { Typography } from 'antd';
import { ColumnType } from 'antd/es/table';

import { getCurrentFreight } from '@ui/features/common/utils';
import { FreightArtifact } from '@ui/features/project/pipelines/freight/freight-artifact';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const versionColumn = (): ColumnType<Stage> => ({
  title: 'Version',
  render: (_, stage) => {
    const currentFreight = getCurrentFreight(stage);

    // TODO: filter by sources
    const firstFreight = currentFreight[0];

    const totalArtifacts =
      (firstFreight?.commits?.length || 0) +
      (firstFreight?.charts?.length || 0) +
      (firstFreight?.images?.length || 0);

    return (
      <>
        {firstFreight?.commits
          ?.slice(0, 2)
          .map((commit) => <FreightArtifact expand key={commit?.repoURL} artifact={commit} />)}

        {firstFreight?.charts
          ?.slice(0, 2)
          .map((chart) => <FreightArtifact expand key={chart?.repoURL} artifact={chart} />)}

        {firstFreight?.images
          ?.slice(0, 2)
          .map((image) => <FreightArtifact expand key={image?.repoURL} artifact={image} />)}

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
      </>
    );
  }
});
