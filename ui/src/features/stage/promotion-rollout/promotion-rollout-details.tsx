import { faArrowLeft, faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Empty, Flex, Tag, Tooltip, Typography, message } from 'antd';
import { format, formatDistanceToNow } from 'date-fns';
import { useMemo } from 'react';
import { Link, generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { LoadingState } from '@ui/features/common/loading-state';
import { getPromotionPhasePresentation } from '@ui/features/common/promotion-status/promotion-phase';
import { RoundProgressBar } from '@ui/features/common/promotion-status/round-progress-bar';
import { SmallLabel } from '@ui/features/common/small-label';
import { getAlias } from '@ui/features/common/utils';
import { getPromotionActor } from '@ui/features/project/pipelines/promotion/get-promotion-actor';
import { useWatchPromotions } from '@ui/features/project/pipelines/promotion/use-watch-promotions';
import { TARGETS_POLL_INTERVAL_MS } from '@ui/features/project/targets/constants';
import {
  useGetPromotionRollout,
  useListPromotions,
  useListTargets,
  usePromoteToStage
} from '@ui/gen/api/v2/core/core';
import { ApiError } from '@ui/lib/api/custom-fetch';
import { parseDate } from '@ui/utils/dates';

import { useGetFreightMap } from '../tabs/freight-history/use-get-freight-map';
import { isPromotionRolloutPhaseTerminal } from '../utils/promotion-rollout';
import {
  freightVersion,
  rolloutDurationMs,
  rolloutPhase,
  targetsTally
} from '../utils/promotion-rollout-list';
import { formatStepDuration } from '../utils/step-duration';

import { ChildPromotions } from './child-promotions';
import { childPromotions, childrenSummary, conditionSummary, firstFailure } from './round-detail';
import { RoundGrid } from './round-grid';
import { WhereItFailed } from './where-it-failed';

// A round that is still fanning out keeps changing. Its children are watched
// and so stay current on their own, but the rollout itself has no single-object
// watch, so it is re-read on a timer for as long as it can still change -- and
// only for as long as reading it works. A failing read left on the timer would
// raise the global error notification every tick, for as long as the tab stays
// open.
const refetchIntervalMs = 5000;

const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div>
    <SmallLabel className='mb-1'>{label}</SmallLabel>
    <span className='text-sm'>{children}</span>
  </div>
);

/**
 * PromotionRolloutDetails is one round of fan-out: the Freight it promotes, how
 * far it got across the Targets it named, where it went wrong if it did, and
 * every child Promotion it produced.
 *
 * The Stage's Promotions tab lists rounds and says nothing about their
 * children; this is where the children are read. Everything a round is
 * measured by -- the bar, the grid, the chips, the "where it failed" panel --
 * is computed from those children rather than from the round's own summary, so
 * no two parts of the view can disagree about what happened.
 */
export const PromotionRolloutDetails = ({
  project,
  stageName,
  promotionRolloutName
}: {
  project: string;
  // The Stage whose tab this is rendered in. It comes from the route rather
  // than from the rollout, so the way back is there before the rollout loads.
  stageName: string;
  promotionRolloutName: string;
}) => {
  const navigate = useNavigate();
  const confirm = useConfirmModal();

  const rolloutQuery = useGetPromotionRollout(project, promotionRolloutName, {
    query: {
      // A round that no longer exists is reported by the view itself, so it
      // needs no notification on top.
      meta: { silent404: true },
      refetchInterval: (query) =>
        query.state.status === 'error' ||
        isPromotionRolloutPhaseTerminal(query.state.data?.data?.status?.phase)
          ? false
          : refetchIntervalMs
    }
  });
  const rollout = rolloutQuery.data?.data;

  const promotionsQuery = useListPromotions(
    project,
    { stage: stageName },
    { query: { enabled: !!stageName } }
  );
  // Targets live in the database, which offers no watch; the list is polled.
  const targetsQuery = useListTargets(
    project,
    { stage: stageName },
    { query: { enabled: !!stageName, refetchInterval: TARGETS_POLL_INTERVAL_MS } }
  );
  useWatchPromotions(project, stageName, !promotionsQuery.isLoading);

  const freightMap = useGetFreightMap(project);
  const freightName = rollout?.spec?.freight || '';
  const freight = freightMap[freightName];

  const children = useMemo(
    () =>
      childPromotions(
        rollout,
        promotionsQuery.data?.data?.items || [],
        targetsQuery.data?.data?.items || []
      ),
    [rollout, promotionsQuery.data, targetsQuery.data]
  );

  const summary = useMemo(() => childrenSummary(children), [children]);
  const tally = targetsTally(summary);
  const failure = useMemo(() => firstFailure(children), [children]);

  const promoteMutation = usePromoteToStage({
    mutation: {
      onSuccess: (data) => {
        // Promoting again starts a round, so that round is where to land. For
        // a target-aware Stage the body is the PromotionRollout the server
        // created, though the generated client types it as a Promotion.
        navigate(
          generatePath(paths.promotionRollout, {
            name: project,
            stageName,
            promotionRolloutName: data.data?.metadata?.name
          })
        );
        message.success(`Promoting ${freightName} to ${stageName} again.`);
      }
    }
  });

  // Back to the list of rounds, which is this same tab without a round named.
  const back = () => navigate(generatePath(paths.stage, { name: project, stageName }));

  const phase = rolloutPhase(rollout);
  const { tagColor, icon, spin } = getPromotionPhasePresentation(phase);
  const started = parseDate(rollout?.status?.startedAt);
  const finished = parseDate(rollout?.status?.finishedAt);
  const duration = rolloutDurationMs(rollout);
  const maxConcurrent = rollout?.spec?.updateStrategy?.maxConcurrent;
  const conditions = conditionSummary(rollout);
  const retryable = ['Failed', 'Errored', 'Aborted'].includes(phase);

  return (
    <Flex vertical gap={16}>
      <Flex justify='space-between' align='flex-start' gap={16} wrap>
        <div>
          <Button
            type='link'
            size='small'
            className='px-0'
            icon={<FontAwesomeIcon icon={faArrowLeft} />}
            onClick={back}
          >
            Promotion Rollouts
          </Button>
          <Flex gap={12} align='center' wrap>
            {/* The Stage is named in the drawer's own header, right above, so
                the title says only what this round promotes. */}
            <Typography.Title level={4} style={{ margin: 0 }}>
              {freightName ? (
                <>
                  Promote {getAlias(freight) || freightName.slice(0, 7)}
                  {freightVersion(freight) && (
                    <Typography.Text type='secondary' className='ml-2 text-sm font-mono'>
                      {freightVersion(freight)}
                    </Typography.Text>
                  )}
                </>
              ) : (
                'Promotion Rollout'
              )}
            </Typography.Title>
            <Tag
              className='m-0'
              color={tagColor}
              icon={<FontAwesomeIcon icon={icon} spin={spin} />}
            >
              {phase}
            </Tag>
          </Flex>
          <Typography.Text type='secondary' className='text-xs font-mono'>
            {rollout?.number ? `#${rollout.number} · ` : ''}
            {promotionRolloutName}
          </Typography.Text>
        </div>
        {retryable && (
          <Button
            icon={<FontAwesomeIcon icon={faUndo} />}
            loading={promoteMutation.isPending}
            onClick={() =>
              confirm({
                title: 'Promote this Freight again?',
                content: (
                  <p>
                    This creates a new rollout promoting the same Freight to every Target the Stage
                    governs now - not only the ones that did not succeed.
                  </p>
                ),
                okText: 'Promote',
                onOk: () =>
                  promoteMutation.mutate({
                    project,
                    stage: stageName,
                    data: { freight: freightName }
                  })
              })
            }
          >
            Retry
          </Button>
        )}
      </Flex>

      {rolloutQuery.isLoading ? (
        <LoadingState />
      ) : !rollout ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          // Only a 404 means the round is gone. Any other failure means it
          // could not be read, and saying it was deleted would be a guess.
          description={
            rolloutQuery.error &&
            !(rolloutQuery.error instanceof ApiError && rolloutQuery.error.isNotFound())
              ? 'This Promotion Rollout could not be read'
              : 'This Promotion Rollout no longer exists'
          }
        />
      ) : (
        <Flex vertical gap={16}>
          <Flex gap={32} wrap>
            <Field label='FREIGHT'>
              {freightName ? (
                <Tooltip title={freightName}>
                  <Link
                    className='font-mono'
                    to={generatePath(paths.freight, { name: project, freightName })}
                  >
                    {getAlias(freight) || freightName.slice(0, 7)}
                  </Link>
                </Tooltip>
              ) : (
                <Typography.Text type='secondary'>none</Typography.Text>
              )}
            </Field>
            <Field label='CREATED BY'>{getPromotionActor(rollout)}</Field>
            <Field label='STARTED'>
              {started ? (
                <Tooltip title={formatDistanceToNow(started, { addSuffix: true })}>
                  {format(started, 'MMM d, HH:mm:ss')}
                </Tooltip>
              ) : (
                <Typography.Text type='secondary'>not yet</Typography.Text>
              )}
            </Field>
            <Field label='FINISHED'>
              {finished ? (
                <>
                  {format(finished, 'HH:mm:ss')}
                  {duration !== undefined && (
                    <Typography.Text type='secondary'>
                      {' '}
                      · {formatStepDuration(duration)}
                    </Typography.Text>
                  )}
                </>
              ) : (
                <Typography.Text type='secondary'>
                  {duration === undefined ? 'not yet' : `running · ${formatStepDuration(duration)}`}
                </Typography.Text>
              )}
            </Field>
            <Field label='UPDATE STRATEGY'>
              {maxConcurrent ? `${maxConcurrent} at a time` : 'all at once'}
            </Field>
          </Flex>

          <Flex gap={16} align='stretch' wrap className='items-stretch'>
            <Flex
              vertical
              gap={12}
              className='flex-1 min-w-80 rounded-md p-3 border border-solid bg-white dark:bg-neutral-800 border-gray-200 dark:border-neutral-700'
            >
              <Flex justify='space-between' align='center' gap={12} wrap>
                <Typography.Text strong className='text-sm'>
                  {tally.total} Promotion{tally.total === 1 ? '' : 's'}
                </Typography.Text>
                {conditions && (
                  <Typography.Text type='secondary' className='text-xs font-mono'>
                    {conditions}
                  </Typography.Text>
                )}
              </Flex>
              <RoundProgressBar summary={summary} size='full' />
              <RoundGrid rows={children} />
              {rollout.status?.message && (
                <Typography.Text type={retryable ? 'danger' : 'secondary'} className='text-xs'>
                  {rollout.status.message}
                </Typography.Text>
              )}
              {failure && (
                <Typography.Text type='secondary' className='text-xs'>
                  First failure: {failure.target}
                  {failure.promotion?.status?.message && ` — ${failure.promotion.status.message}`}
                </Typography.Text>
              )}
            </Flex>

            <div className='w-72'>
              <WhereItFailed rows={children} />
            </div>
          </Flex>

          <ChildPromotions
            project={project}
            rows={children}
            loading={promotionsQuery.isLoading || targetsQuery.isLoading}
          />
        </Flex>
      )}
    </Flex>
  );
};
