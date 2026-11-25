import { Button, Flex, Typography } from 'antd';

import { PhaseFilter, PromotionPhase } from './promotion-phase';

/**
 * PhaseFilterChips narrows a list to one phase: All, then one chip per phase
 * worth offering, each with how many of the list are in it.
 *
 * A chip nothing is in cannot be picked, so the list it governs never empties
 * with no explanation, and All is always present rather than making a reader
 * discover that the way back is to click the selected chip again.
 *
 * A chip is named by its phase unless `labelOf` names it otherwise, for a list
 * whose phases are not all spelled the way they are read.
 */
export const PhaseFilterChips = <P extends string = PromotionPhase>({
  filters,
  selected,
  onSelect,
  labelOf
}: {
  filters: PhaseFilter<P>[];
  selected?: P;
  onSelect: (phase?: P) => void;
  labelOf?: (phase: P) => string;
}) => (
  <Flex gap={8} wrap>
    {filters.map(({ phase, count }) => (
      <Button
        key={phase ?? 'all'}
        size='small'
        shape='round'
        type={selected === phase ? 'primary' : 'default'}
        disabled={!!phase && count === 0}
        aria-pressed={selected === phase}
        onClick={() => onSelect(phase)}
      >
        {phase ? (labelOf?.(phase) ?? phase) : 'All'}{' '}
        <Typography.Text type='secondary' className='text-xs'>
          {count}
        </Typography.Text>
      </Button>
    ))}
  </Flex>
);
