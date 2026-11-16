import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Table, Tag, Tooltip, Typography, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format, formatDistanceToNow } from 'date-fns';
import { ReactNode, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { PhaseFilterChips } from '@ui/features/common/promotion-status/phase-filter-chips';
import {
  PromotionPhase,
  getPromotionPhasePresentation
} from '@ui/features/common/promotion-status/promotion-phase';
import { RoundProgressBar } from '@ui/features/common/promotion-status/round-progress-bar';
import { getAlias } from '@ui/features/common/utils';
import { getPromotionActor } from '@ui/features/project/pipelines/promotion/get-promotion-actor';
import { parseDate } from '@ui/utils/dates';

import { RoundIdentityLabel } from './round-identity-label';
import { RoundKind, roundWords } from './round-words';
import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import {
  describeTargets,
  freightVersion,
  phaseFilters,
  requestDurationMs,
  requestPhase,
  roundSummary,
  targetsTally
} from './utils/promotion-request-list';
import { Round, isRolloutRound, roundIdentity, roundMessage } from './utils/promotion-rollout';
import { formatStepDuration } from './utils/step-duration';

const roundsPagination = { hideOnSinglePage: true, pageSize: 10, showSizeChanger: false };

// TargetsCell draws how a round fared: the same bar every other surface draws a
// round with, and beneath it the figure per phase. Only what needs to stand out
// is painted -- a failure in red, movement in blue. Succeeded is the expected
// case and reads as ordinary text, and Pending and Aborted are painted with
// fills too faint to read as text at all.
const TargetsCell = ({ round, kind }: { round: Round; kind: RoundKind }) => {
  const { token } = theme.useToken();
  const summary = roundSummary(round);
  const tally = targetsTally(summary);

  const partColor = (phase: PromotionPhase) => {
    if (phase === 'Failed' || phase === 'Errored') {
      return token.colorError;
    }
    return phase === 'Running' ? token.colorInfo : token.colorTextSecondary;
  };

  if (!tally.total) {
    return (
      <Tooltip
        title={`The Stage governed no Targets when this ${roundWords[kind].short} was created`}
      >
        <Typography.Text type='secondary' className='text-xs'>
          no Targets
        </Typography.Text>
      </Tooltip>
    );
  }

  return (
    <Flex vertical gap={6} className='min-w-0'>
      <RoundProgressBar summary={summary} />
      <span className='text-xs' aria-label={describeTargets(tally)}>
        {tally.parts.map(({ phase, count }, index) => (
          <span key={phase} style={{ color: partColor(phase) }}>
            {index > 0 && <span style={{ color: token.colorTextQuaternary }}> · </span>}
            {count} {phase.toLowerCase()}
          </span>
        ))}
        <Typography.Text type='secondary'> / {tally.total}</Typography.Text>
      </span>
    </Flex>
  );
};

type Props = {
  project: string;
  // Which of the two these rounds are, for what the table calls them.
  kind: RoundKind;
  // The rounds to list, newest first.
  rounds: Round[];
  loading: boolean;
  // Where a round is read. A round with nowhere to lead is listed unlinked.
  pathOf: (round: Round) => string | undefined;
  // Shown beneath the table, for a list that can read on.
  footer?: ReactNode;
};

/**
 * RoundsTable lists the rounds of promotion a target-aware Stage has run,
 * newest first.
 *
 * Such a Stage promotes Freight to the Targets it governs by way of a round --
 * a PromotionRollout, or a PromotionRequest where rollouts are not in use --
 * which fans out into one child Promotion per Target. A fleet of any size
 * therefore produces hundreds of Promotions that say nothing individually, so
 * this level shows only the rounds -- how far each one got and what it was
 * promoting -- and the children are read inside a round.
 */
export const RoundsTable = ({ project, kind, rounds, loading, pathOf, footer }: Props) => {
  const navigate = useNavigate();
  const words = roundWords[kind];

  const freightMap = useGetFreightMap(project);

  const [selectedPhase, setSelectedPhase] = useState<PromotionPhase | undefined>();

  const rows = useMemo(
    () =>
      selectedPhase ? rounds.filter((round) => requestPhase(round) === selectedPhase) : rounds,
    [rounds, selectedPhase]
  );

  const columns: ColumnsType<Round> = [
    {
      title: 'Phase',
      width: 120,
      render: (_, round) => {
        const phase = requestPhase(round);
        const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
        return (
          <Tag className='m-0' color={tagColor} icon={<FontAwesomeIcon icon={icon} spin={spin} />}>
            {phase}
          </Tag>
        );
      }
    },
    {
      title: 'Freight',
      render: (_, round) => {
        const freightName = round?.spec?.freight || '';
        const freight = freightMap[freightName];
        const version = freightVersion(freight);
        // A round that did not succeed explains itself in status.message, and
        // that explanation is the reason a reader opened the tab. Whether one
        // that did succeed has anything to add is roundMessage's call.
        const message = roundMessage(round);
        // A rollout's number leads the line, set back like the version that
        // ends it: "#4 vigilant-numbat 1.27.2". A request's name is too long
        // for that and keeps a line of its own.
        const numbered = isRolloutRound(round);
        const failed = ['Failed', 'Errored'].includes(requestPhase(round));
        const path = pathOf(round);
        const label = getAlias(freight) || freightName.slice(0, 7);
        return (
          <Flex vertical gap={2} className='min-w-0'>
            <span>
              {numbered && (
                <RoundIdentityLabel {...roundIdentity(round)} className='text-xs mr-2' />
              )}
              <Tooltip title={freightName}>
                <span>
                  {path ? (
                    <Link className='font-semibold' to={path}>
                      {label}
                    </Link>
                  ) : (
                    <span className='font-semibold'>{label}</span>
                  )}
                  {version && (
                    <Typography.Text type='secondary' className='ml-2 text-xs font-mono'>
                      {version}
                    </Typography.Text>
                  )}
                </span>
              </Tooltip>
            </span>
            {!numbered && (
              <RoundIdentityLabel {...roundIdentity(round)} className='text-xs self-start' />
            )}
            {message && (
              <Typography.Text type={failed ? 'danger' : 'secondary'} className='text-xs'>
                {message}
              </Typography.Text>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Targets',
      width: 230,
      render: (_, round) => <TargetsCell round={round} kind={kind} />
    },
    {
      title: 'Started',
      width: 170,
      render: (_, round) => {
        // A round the reconciler has not started yet has no startedAt, but it
        // does have a creation time, which is when the user asked for it.
        const started =
          parseDate(round?.status?.startedAt) || parseDate(round?.metadata?.creationTimestamp);
        if (!started) {
          return null;
        }
        return (
          <Flex vertical gap={2}>
            <Typography.Text className='text-xs'>
              {format(started, 'MMM d, HH:mm:ss')}
            </Typography.Text>
            <Typography.Text type='secondary' className='text-xs'>
              {formatDistanceToNow(started, { addSuffix: true })}
            </Typography.Text>
          </Flex>
        );
      }
    },
    {
      title: 'Duration',
      width: 110,
      render: (_, round) => {
        const duration = requestDurationMs(round);
        return (
          <Typography.Text
            className='text-xs tabular-nums'
            type={duration === undefined ? 'secondary' : undefined}
          >
            {duration === undefined ? '-' : formatStepDuration(duration)}
          </Typography.Text>
        );
      }
    },
    {
      title: 'By',
      width: 110,
      render: (_, round) => (
        <Typography.Text className='text-xs'>{getPromotionActor(round)}</Typography.Text>
      )
    }
  ];

  return (
    <Flex vertical gap={12}>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Typography.Title level={5} className='m-0'>
          {words.many}
        </Typography.Title>
        {rounds.length > 0 && (
          <PhaseFilterChips
            filters={phaseFilters(rounds)}
            selected={selectedPhase}
            onSelect={setSelectedPhase}
          />
        )}
      </Flex>

      <Table
        dataSource={rows}
        columns={columns}
        size='small'
        pagination={roundsPagination}
        onRow={(round) => ({
          className: pathOf(round) ? 'cursor-pointer' : undefined,
          onClick: (event) => {
            const path = pathOf(round);
            // The Freight cell is a real link, so that it can be opened in a
            // new tab and read before it is followed. Let it handle its own
            // click rather than navigating twice.
            if (!path || (event.target as HTMLElement).closest('a')) {
              return;
            }
            navigate(path);
          }
        })}
        rowKey={(round) => round?.metadata?.name || ''}
        loading={loading}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description='No Freight has been promoted to this Stage yet'
            />
          )
        }}
      />

      {footer}

      {rounds.length > 0 && (
        <Typography.Text type='secondary' className='text-xs'>
          Each {words.short} fans out to one Promotion per Target. Open a {words.short} to see its
          Promotions.
        </Typography.Text>
      )}
    </Flex>
  );
};
