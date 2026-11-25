import { faArrowRight, faFileCode, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tag, Typography, theme } from 'antd';
import { format } from 'date-fns';
import { ReactNode } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useModal } from '@ui/features/common/modal/use-modal';
import { TargetYamlModal } from '@ui/features/project/targets/detail/target-yaml-modal';
import { flattenParams } from '@ui/features/project/targets/model/target-details-utils';
import { visibleLabels } from '@ui/features/project/targets/model/target-labels';
import { TargetRow, governingOrigins } from '@ui/features/project/targets/model/target-rows';
import {
  TargetState,
  targetStatePresentations
} from '@ui/features/project/targets/model/target-state';
import { LastPromotion } from '@ui/features/project/targets/shared/last-promotion';
import { OriginRows, Row } from '@ui/features/project/targets/shared/origin-rows';
import { failedStepName } from '@ui/features/stage/promotion-request/round-detail';
import { rolloutPath } from '@ui/features/stage/utils/promotion-rollout';
import { useGetPromotion } from '@ui/gen/api/v2/core/core';
import { parseDate } from '@ui/utils/dates';

import { governing, owningTargetSet } from './summary';

const cardClassName =
  'flex-1 min-w-[240px] rounded-md p-3 border border-solid bg-gray-50 dark:bg-neutral-900 ' +
  'border-gray-200 dark:border-neutral-700';

const Card = ({ heading, children }: { heading: string; children: ReactNode }) => (
  <div className={cardClassName}>
    <div className='text-xs font-semibold mb-2'>{heading}</div>
    <Flex vertical gap={6}>
      {children}
    </Flex>
  </div>
);

type Headline = {
  text: string;
  color: 'default' | 'processing' | 'success' | 'warning' | 'error';
};

// headline says where this Target stands with this Stage, whatever the grid
// happens to be coloured by: a reader who picked a square wants the Target's
// story, not a restatement of the colour they clicked. The words are the ones
// every Targets surface uses for the state.
const headlineColors: Record<TargetState, Headline['color']> = {
  UpToDate: 'success',
  Behind: 'warning',
  Promoting: 'processing',
  Pending: 'default',
  Failed: 'error',
  NotAttempted: 'default',
  NeverPromoted: 'default'
};

const headline = (state: TargetState): Headline => ({
  text: state === 'Failed' ? 'Last promotion failed' : targetStatePresentations[state].label,
  color: headlineColors[state]
});

/**
 * TargetFacts answers the question a picked square asks: what is this one
 * Target, what is it running for this Stage, how its last Promotion from the
 * Stage went, and -- when that went wrong -- what went wrong and where to read
 * it.
 *
 * It sits under the grid rather than in a drawer so the fleet stays on screen:
 * picking one square after another reads across a group without anything
 * opening and closing over it.
 */
export const TargetFacts = ({
  project,
  stageName,
  row,
  freightLabel,
  onClose
}: {
  project: string;
  stageName: string;
  row: TargetRow;
  freightLabel: (name: string) => string;
  onClose: () => void;
}) => {
  const { token } = theme.useToken();
  const { show } = useModal();

  const target = row.target;
  const name = target.metadata?.name || '';
  const governed = governing(row);
  const state = governed?.state || 'NeverPromoted';
  const last = governed?.entry.lastPromotion;
  const failed = state === 'Failed';

  // The status names the last Promotion; the Promotion is what carries the
  // step it stopped on. Fetched only for a Target being asked about, and only
  // when there is a failure to explain.
  const promotion = last?.name;
  const promotionQuery = useGetPromotion(project, promotion || '', {
    query: { enabled: failed && !!promotion }
  });
  const failedStep = failedStepName(promotionQuery.data?.data);
  // A failure is read on the rollout that made the Promotion, where there is
  // one, and on the Promotion itself otherwise.
  const failureLink = last?.rollout
    ? { to: rolloutPath(project, stageName, last.rollout), kind: 'rollout' }
    : promotion
      ? {
          to: generatePath(paths.promotion, { name: project, promotionId: promotion }),
          kind: 'promotion'
        }
      : undefined;

  const origins = governingOrigins(governed);
  const anyBehind = origins.some(({ behind }) => behind);
  const params = flattenParams(target.spec?.params);
  const targetSet = owningTargetSet(target);
  const created = parseDate(target.metadata?.creationTimestamp);
  const { text, color } = headline(state);

  return (
    <div className='mt-3 rounded-md p-4 border border-solid border-gray-200 dark:border-neutral-700'>
      <Flex justify='space-between' align='flex-start' gap={12} wrap>
        <Flex gap={8} align='center' wrap>
          <span className='font-mono font-semibold'>{name}</span>
          <Tag className='m-0' color={color}>
            {text}
          </Tag>
          {/* Health only when something has assessed it; an unassessed Target
              is not in a state of its own. */}
          {governed?.health?.status && (
            <Tag className='m-0'>Health {governed.health.status.toLowerCase()}</Tag>
          )}
        </Flex>
        <Flex gap={8} align='center'>
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faFileCode} />}
            onClick={() => show((p) => <TargetYamlModal {...p} target={target} />)}
          >
            YAML
          </Button>
          <Button
            size='small'
            type='text'
            aria-label='Close'
            icon={<FontAwesomeIcon icon={faXmark} />}
            onClick={onClose}
          />
        </Flex>
      </Flex>

      <Flex gap={6} align='center' wrap className='mt-2'>
        {visibleLabels(target).map(([key, value]) => (
          <Tag key={key} className='m-0 text-xs font-mono'>
            {key}={value}
          </Tag>
        ))}
        {targetSet && (
          <Typography.Text type='secondary' className='text-xs'>
            · from TargetSet <span className='font-mono'>{targetSet}</span>
          </Typography.Text>
        )}
        {created && (
          <Typography.Text type='secondary' className='text-xs'>
            · created {format(created, 'MMM d, HH:mm')}
          </Typography.Text>
        )}
      </Flex>

      <Flex gap={12} wrap className='mt-3'>
        <Card heading={`In ${stageName}`}>
          {origins.length ? (
            <OriginRows project={project} origins={origins} freightLabel={freightLabel} />
          ) : (
            <Typography.Text type='secondary' className='text-xs'>
              Nothing promoted here yet
            </Typography.Text>
          )}

          {/* Each origin's own line says when it is behind. A Stage that has
              moved on with no origin to show for it still needs saying. */}
          {governed?.behind && !anyBehind && (
            <Typography.Text type='warning' className='text-xs'>
              Behind: Stage is on other Freight
            </Typography.Text>
          )}

          {governed && (
            <Row label='Last promotion'>
              <LastPromotion project={project} entry={governed.entry} freightLabel={freightLabel} />
            </Row>
          )}

          {failed && (
            <div className='text-xs' style={{ color: token.colorError }}>
              Promoting {freightLabel(last?.freight || '')}{' '}
              {(last?.phase || 'failed').toLowerCase()}
              {failedStep && ` at step ${failedStep}`}
              {failureLink && (
                <>
                  {' · '}
                  <Link style={{ color: token.colorError }} to={failureLink.to}>
                    view {failureLink.kind}
                    <FontAwesomeIcon icon={faArrowRight} className='ml-1' />
                  </Link>
                </>
              )}
            </div>
          )}
        </Card>

        <Card heading='Params'>
          {params.length ? (
            params.map((param) => (
              <Row key={param.key} label={param.key}>
                <span className='font-mono'>{param.value}</span>
              </Row>
            ))
          ) : (
            <Typography.Text type='secondary' className='text-xs'>
              This Target carries no params
            </Typography.Text>
          )}
        </Card>
      </Flex>
    </div>
  );
};
