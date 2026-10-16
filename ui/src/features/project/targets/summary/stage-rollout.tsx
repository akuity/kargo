import { Flex, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorMap } from '@ui/features/stage/utils';

import { StageRollout } from '../model/target-rows';
import { FreightLink } from '../shared/freight-link';
import { StateBar } from '../shared/state-visuals';

// rolloutCaption says in words what the bar above it draws: how much of the
// Stage's fleet has caught up, and what is wrong or still moving.
const rolloutCaption = (rollout: StageRollout): string => {
  if (!rollout.governed) {
    return 'no Targets';
  }
  const parts = [
    ...(rollout.inFlight ? ['Rolling out'] : []),
    `${rollout.onLatest}/${rollout.governed} on latest`,
    ...(rollout.unhealthy ? [`${rollout.unhealthy} unhealthy`] : []),
    ...(rollout.inFlight ? [`${rollout.inFlight} in flight`] : [])
  ];
  return parts.join(' · ');
};

const StageCard = ({
  project,
  rollout,
  stageColorMap,
  freightLabel
}: {
  project: string;
  rollout: StageRollout;
  stageColorMap: ColorMap;
  freightLabel: (name: string) => string;
}) => {
  const name = rollout.stage.metadata?.name || '';
  return (
    <div className='rounded-md px-3 py-2.5 border border-solid bg-white dark:bg-neutral-800 border-gray-200 dark:border-neutral-700'>
      <Flex justify='space-between' align='center' gap={8}>
        <Link to={generatePath(paths.stage, { name: project, stageName: name })}>
          <Flex gap={6} align='center' className='min-w-0'>
            <span
              className='inline-block shrink-0 rounded-sm w-2 h-2 bg-gray-500'
              style={{ backgroundColor: stageColorMap[name] }}
            />
            <span className='text-sm font-semibold truncate'>{name}</span>
          </Flex>
        </Link>
        <Flex gap={6} align='center'>
          {rollout.freight.map((freight) => (
            <FreightLink
              key={freight}
              project={project}
              name={freight}
              label={freightLabel(freight)}
              className='text-xs'
            />
          ))}
        </Flex>
      </Flex>
      <StateBar counts={rollout.counts} className='my-2' />
      <Typography.Text type='secondary' className='text-xs'>
        {rolloutCaption(rollout)}
      </Typography.Text>
    </div>
  );
};

/**
 * StageRollouts shows how far each Stage has got in reaching the Targets it
 * governs: the Freight it is on, a bar of its fleet's states, and how much of
 * that fleet has caught up with it. Read across the row and a promotion moving
 * down the pipeline is visible as it happens.
 */
export const StageRollouts = ({
  project,
  rollouts,
  stageColorMap,
  freightLabel
}: {
  project: string;
  rollouts: StageRollout[];
  stageColorMap: ColorMap;
  freightLabel: (name: string) => string;
}) => (
  <div
    className='grid gap-3'
    style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(210px, 1fr))' }}
  >
    {rollouts.map((rollout) => (
      <StageCard
        key={rollout.stage.metadata?.name}
        project={project}
        rollout={rollout}
        stageColorMap={stageColorMap}
        freightLabel={freightLabel}
      />
    ))}
  </div>
);
