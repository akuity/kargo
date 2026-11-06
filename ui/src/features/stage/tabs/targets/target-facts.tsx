import { faArrowRight, faFileCode, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tag, Typography, theme } from 'antd';
import { format } from 'date-fns';
import { ReactNode } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useModal } from '@ui/features/common/modal/use-modal';
import { TargetYamlModal } from '@ui/features/project/targets/detail/target-yaml-modal';
import { freightByOrigin } from '@ui/features/project/targets/model/round-utils';
import { flattenParams } from '@ui/features/project/targets/model/target-details-utils';
import { visibleLabels } from '@ui/features/project/targets/model/target-labels';
import { TargetRow } from '@ui/features/project/targets/model/target-rows';
import { OriginRows, Row } from '@ui/features/project/targets/shared/origin-rows';
import { failedStepName } from '@ui/features/stage/promotion-request/round-detail';
import { useGetPromotion } from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { childPromotionName, governing, owningTargetSet } from './summary';

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

type Headline = { text: string; color: 'default' | 'processing' | 'success' | 'error' };

// headline says what last happened to this Target under this Stage, whatever
// the grid happens to be coloured by: a reader who picked a square wants the
// Target's story, not a restatement of the colour they clicked.
const headline = (phase: string | undefined, promoted: boolean, blocked: boolean): Headline => {
  // A blocked round never fanned out, so it did nothing here either way.
  if (blocked) {
    return { text: 'Not attempted', color: 'default' };
  }
  switch (phase) {
    case 'Failed':
    case 'Errored':
      return { text: 'Last promotion failed', color: 'error' };
    case 'Succeeded':
      return { text: 'Last promotion succeeded', color: 'success' };
    case 'Running':
    case 'Pending':
      return { text: 'Promotion in flight', color: 'processing' };
    case 'Aborted':
      return { text: 'Promotion aborted', color: 'default' };
    default:
      return promoted
        ? { text: 'Not in latest round', color: 'default' }
        : { text: 'Never promoted', color: 'default' };
  }
};

/**
 * TargetFacts answers the question a picked square asks: what is this one
 * Target, what is it running for this Stage, and -- when the Stage's latest
 * round went wrong here -- what went wrong and where to read it.
 *
 * It sits under the grid rather than in a drawer so the fleet stays on screen:
 * picking one square after another reads across a group without anything
 * opening and closing over it.
 */
export const TargetFacts = ({
  project,
  stageName,
  row,
  round,
  blocked = false,
  freightLabel,
  onClose
}: {
  project: string;
  stageName: string;
  row: TargetRow;
  // The Stage's latest round, which names the child Promotion this Target got.
  round?: PromotionRequest;
  // Whether that round is blocked, and so reached nothing.
  blocked?: boolean;
  freightLabel: (name: string) => string;
  onClose: () => void;
}) => {
  const { token } = theme.useToken();
  const { show } = useModal();

  const target = row.target;
  const name = target.metadata?.name || '';
  const governed = governing(row);
  const phase = governed?.phase;
  const failed = phase === 'Failed' || phase === 'Errored';

  // The round records a phase per Target; the child Promotion is what carries
  // the step it stopped on. Fetched only for a Target being asked about, and
  // only when there is a failure to explain.
  const promotion = childPromotionName(round, name);
  const promotionQuery = useGetPromotion(project, promotion || '', {
    query: { enabled: failed && !!promotion }
  });
  const failedStep = failedStepName(promotionQuery.data?.data);

  const currentFreight = governed?.currentFreight;
  // A Target with no Freight is only shown what the Stage is on once a
  // promotion to it has failed. Until then it has simply not been promoted to,
  // and is not behind on anything.
  const origins =
    currentFreight || governed?.upToDate === false
      ? freightByOrigin(currentFreight, governed?.stage.status?.freightHistory?.[0])
      : [];
  const anyBehind = origins.some(({ behind }) => behind);
  const params = flattenParams(target.spec?.params);
  const targetSet = owningTargetSet(target);
  const created = parseDate(target.metadata?.creationTimestamp);
  const { text, color } = headline(phase, !!row.freight.length, blocked);

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
          {governed?.upToDate === false && !anyBehind && (
            <Typography.Text type='warning' className='text-xs'>
              Behind: Stage is on other Freight
            </Typography.Text>
          )}

          {failed && (
            <div className='text-xs' style={{ color: token.colorError }}>
              Promoting {freightLabel(round?.spec?.freight || '')} failed
              {failedStep && ` at step ${failedStep}`}
              {round?.metadata?.name && (
                <>
                  {' · '}
                  <Link
                    style={{ color: token.colorError }}
                    to={generatePath(paths.promotionRequest, {
                      name: project,
                      stageName,
                      promotionRequestName: round.metadata.name
                    })}
                  >
                    view request
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
