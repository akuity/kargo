import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tag, Tooltip, Typography } from 'antd';
import { formatDistanceToNow } from 'date-fns';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  getPromotionPhasePresentation,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { RoundProgressBar } from '@ui/features/common/promotion-status/round-progress-bar';
import { SmallLabel } from '@ui/features/common/small-label';
import { PromotionRequest, PromotionRequestSummary } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

// PhaseChips renders one chip per phase present in a round, in the order the
// progress bar draws them, so a reader sees every phase's count and not only
// the succeeded figure. Rounds with trouble keep their failures visible even
// when they are a sliver of the bar.
//
// With onSelect, the chips double as a phase filter: clicking one selects it,
// clicking it again clears the selection.
export const PhaseChips = ({
  summary,
  selected,
  onSelect
}: {
  summary: PromotionRequestSummary;
  selected?: string;
  onSelect?: (phase?: string) => void;
}) => (
  <Flex gap={4} wrap>
    {promotionPhases.map((phase) => {
      const count = summary[phase.toLowerCase() as keyof PromotionRequestSummary] || 0;
      if (!count) {
        return null;
      }
      const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
      const description = `${count} ${phase.toLowerCase()} Target${count === 1 ? '' : 's'}`;
      const active = selected === phase;
      return (
        <Tooltip key={phase} title={onSelect ? `Show only ${phase.toLowerCase()}` : description}>
          <Tag
            className={onSelect ? 'm-0 cursor-pointer' : 'm-0'}
            color={tagColor}
            icon={<FontAwesomeIcon icon={icon} spin={spin} />}
            aria-label={description}
            aria-pressed={onSelect ? active : undefined}
            style={onSelect && !active && selected ? { opacity: 0.5 } : undefined}
            onClick={onSelect ? () => onSelect(active ? undefined : phase) : undefined}
          >
            {count} {phase.toLowerCase()}
          </Tag>
        </Tooltip>
      );
    })}
  </Flex>
);

// RoundCard summarizes a Stage's latest round of fan-out as a card in the
// style of the Requested Freight cards above it: which Freight the round
// promotes, the round's phase, when it finished, how many Targets succeeded,
// and a bar across the foot of the card drawn from the same counts as the
// rows beneath it.
export const RoundCard = ({
  projectName,
  round,
  summary,
  freightLabel,
  showChips = true
}: {
  projectName: string;
  round: PromotionRequest;
  summary: PromotionRequestSummary;
  freightLabel: (name: string) => string;
  // showChips puts the per-phase chips on the card. Turn it off when the
  // same chips appear directly beneath the card as a filter.
  showChips?: boolean;
}) => {
  const freight = round.spec?.freight || '';
  const phase = round.status?.phase || 'Pending';
  const finished = parseDate(round.status?.finishedAt);
  const started = parseDate(round.status?.startedAt);
  const total = Object.values(summary).reduce((sum, count) => sum + (count || 0), 0);
  const succeeded = summary.succeeded || 0;

  return (
    <div className='bg-gray-50 dark:bg-neutral-800 rounded-md p-3 border-2 border-solid border-gray-200 dark:border-neutral-700'>
      <Flex gap={32} align='flex-start' wrap className='mb-3'>
        <div>
          <SmallLabel className='mb-1'>LATEST ROUND</SmallLabel>
          {freight ? (
            <Link
              className='font-semibold'
              to={generatePath(paths.freight, { name: projectName, freightName: freight })}
            >
              {freightLabel(freight)}
            </Link>
          ) : (
            <Typography.Text type='secondary'>none</Typography.Text>
          )}
          {round.metadata?.name && (
            <Tooltip title='The PromotionRequest this round is. Its Promotions tab row lists every child Promotion.'>
              <Typography.Text type='secondary' className='block text-xs font-mono mt-0.5'>
                {round.metadata.name}
              </Typography.Text>
            </Tooltip>
          )}
        </div>
        <div>
          <SmallLabel className='mb-1'>PHASE</SmallLabel>
          <Flex gap={6} align='center'>
            <PromotionStatusIcon subject='Promotion Request' status={round.status} />
            <span className='text-sm'>{phase}</span>
          </Flex>
        </div>
        <div>
          <SmallLabel className='mb-1'>{finished ? 'FINISHED' : 'STARTED'}</SmallLabel>
          <span className='text-sm'>
            {finished
              ? formatDistanceToNow(finished, { addSuffix: true })
              : started
                ? formatDistanceToNow(started, { addSuffix: true })
                : 'not yet'}
          </span>
        </div>
        <div className='ml-auto text-right'>
          <SmallLabel className='mb-1'>TARGETS</SmallLabel>
          <span className='text-sm'>
            <span className='font-semibold'>{succeeded}</span>
            <Typography.Text type='secondary'> of {total} succeeded</Typography.Text>
          </span>
          {showChips && (
            <Flex justify='flex-end' className='mt-2'>
              <PhaseChips summary={summary} />
            </Flex>
          )}
        </div>
      </Flex>
      <RoundProgressBar summary={summary} size='compact' />
    </div>
  );
};
