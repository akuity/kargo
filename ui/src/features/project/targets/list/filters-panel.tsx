import { Button, Checkbox, Flex, Typography } from 'antd';

import {
  Facets,
  LabelFacet,
  STATE_FACET,
  facetExclusions,
  noFacets,
  toggleFacet
} from '../model/target-facets';
import { StateCounts, statePrecedence, targetStatePresentations } from '../model/target-state';
import { StateDot } from '../shared/state-visuals';

const Entry = ({
  checked,
  label,
  count,
  onChange
}: {
  checked: boolean;
  label: React.ReactNode;
  count: number;
  onChange: () => void;
}) => (
  <Flex align='center' gap={8}>
    <Checkbox checked={checked} onChange={onChange} />
    <span className='text-xs min-w-0 truncate flex-1 cursor-pointer' onClick={onChange}>
      {label}
    </span>
    <span className='text-xs text-gray-500 dark:text-neutral-400 shrink-0'>{count}</span>
  </Flex>
);

/**
 * FiltersPanel narrows the fleet by what its Targets are labelled with and by
 * where they stand. Everything starts ticked, so the panel reads as a census
 * of the fleet first and a filter second: unticking a value takes it out.
 *
 * Each figure counts the Targets that value would bring back, so it does not
 * fall to nothing the moment the value is unticked.
 */
export const FiltersPanel = ({
  labels,
  states,
  facets,
  onChange
}: {
  labels: LabelFacet[];
  states: StateCounts;
  facets: Facets;
  onChange: (facets: Facets) => void;
}) => {
  const exclusions = facetExclusions(facets);
  const toggle = (key: string, value: string) => onChange(toggleFacet(facets, key, value));

  return (
    <Flex
      vertical
      gap={16}
      className='rounded-md p-4 border border-solid bg-white dark:bg-neutral-800 border-gray-200 dark:border-neutral-700'
    >
      <Flex justify='space-between' align='center'>
        <Typography.Text strong className='text-sm'>
          Status
        </Typography.Text>
        {/* Always rendered, disabled when there is nothing to reset: appearing
            and disappearing shifts the whole panel as filters are picked. */}
        <Button
          type='link'
          size='small'
          className='px-0'
          disabled={exclusions === 0}
          onClick={() => onChange(noFacets())}
        >
          Reset
        </Button>
      </Flex>

      <Flex vertical gap={6}>
        {statePrecedence
          // A state switched off keeps its box even once its tally is zero --
          // another facet may have emptied it, and it is still filtering.
          .filter((state) => states[state] || facets.states.includes(state))
          .map((state) => (
            <Entry
              key={state}
              checked={!facets.states.includes(state)}
              label={
                <Flex gap={6} align='center'>
                  <StateDot state={state} />
                  {targetStatePresentations[state].label}
                </Flex>
              }
              count={states[state]}
              onChange={() => toggle(STATE_FACET, state)}
            />
          ))}
      </Flex>

      {labels.length > 0 && (
        <Typography.Text
          strong
          className='text-sm border-0 border-t border-solid border-gray-200 dark:border-neutral-700 pt-4'
        >
          Labels
        </Typography.Text>
      )}

      {labels.map((facet) => (
        <Flex key={facet.key} vertical gap={6}>
          <Typography.Text type='secondary' className='text-xs font-mono'>
            {facet.key}
          </Typography.Text>
          {facet.values.map(({ value, count, excluded }) => (
            <Entry
              key={value}
              checked={!excluded}
              label={value}
              count={count}
              onChange={() => toggle(facet.key, value)}
            />
          ))}
        </Flex>
      ))}
    </Flex>
  );
};
