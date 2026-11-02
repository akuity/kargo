import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Checkbox, Flex, Input, Typography } from 'antd';
import { useMemo, useState } from 'react';

import {
  Facets,
  LabelFacet,
  STATE_FACET,
  facetSelections,
  noFacets,
  searchLabelFacets,
  toggleFacet,
  truncateFacetValues
} from '../model/target-facets';
import { StateCounts, stateDisplayOrder, targetStatePresentations } from '../model/target-state';

import { StateDot } from './state-visuals';

// A label key lists this many values before the rest fold behind a link. A
// key like hostname may carry one value per Target.
const valueLimit = 5;

// The label search is offered once the fleet carries more label values than
// this. A few values are quicker to read than to search.
const searchThreshold = 15;

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
 * where they stand. Nothing starts ticked, and nothing ticked filters nothing:
 * ticking a value narrows the fleet to it, and ticking more values of the same
 * facet widens it to any of them.
 *
 * Each figure counts the Targets that value would bring in, so the values not
 * yet ticked do not fall to nothing the moment another one is.
 */
export const FiltersPanel = ({
  labels,
  states,
  facets,
  labelValueCount,
  onChange
}: {
  labels: LabelFacet[];
  states: StateCounts;
  facets: Facets;
  // How many label values the whole fleet carries, unfiltered.
  labelValueCount: number;
  onChange: (facets: Facets) => void;
}) => {
  const selections = facetSelections(facets);
  const toggle = (key: string, value: string) => onChange(toggleFacet(facets, key, value));

  const [search, setSearch] = useState('');
  const shownLabels = useMemo(() => searchLabelFacets(labels, search), [labels, search]);
  // A search already typed keeps its box, so it can always be cleared.
  const searchable = labelValueCount > searchThreshold || search !== '';
  // Label keys the reader has opened past the first few values.
  const [expanded, setExpanded] = useState<string[]>([]);
  const toggleExpanded = (key: string) =>
    setExpanded(
      expanded.includes(key) ? expanded.filter((held) => held !== key) : [...expanded, key]
    );

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
          disabled={selections === 0}
          onClick={() => onChange(noFacets())}
        >
          Reset
        </Button>
      </Flex>

      <Flex vertical gap={6}>
        {stateDisplayOrder
          // A picked state keeps its box even once its tally is zero --
          // another facet may have emptied it, and it is still filtering.
          .filter((state) => states[state] || facets.states.includes(state))
          .map((state) => (
            <Entry
              key={state}
              checked={facets.states.includes(state)}
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
        <Flex
          vertical
          gap={8}
          className='border-0 border-t border-solid border-gray-200 dark:border-neutral-700 pt-4'
        >
          <Typography.Text strong className='text-sm'>
            Labels
          </Typography.Text>
          {searchable && (
            <Input
              allowClear
              size='small'
              prefix={<FontAwesomeIcon icon={faMagnifyingGlass} className='text-gray-400' />}
              placeholder='Search labels'
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          )}
        </Flex>
      )}

      {labels.length > 0 && shownLabels.length === 0 && (
        <Typography.Text type='secondary' className='text-xs'>
          No labels match
        </Typography.Text>
      )}

      {shownLabels.map((facet) => {
        const open = expanded.includes(facet.key);
        const { shown, hidden } = open
          ? { shown: facet.values, hidden: 0 }
          : truncateFacetValues(facet.values, valueLimit);
        return (
          <Flex key={facet.key} vertical gap={6}>
            <Typography.Text type='secondary' className='text-xs font-mono'>
              {facet.key}
            </Typography.Text>
            {shown.map(({ value, count, selected }) => (
              <Entry
                key={value}
                checked={selected}
                label={value}
                count={count}
                onChange={() => toggle(facet.key, value)}
              />
            ))}
            {(hidden > 0 || open) && facet.values.length > valueLimit && (
              <Button
                type='link'
                size='small'
                className='px-0 self-start text-xs h-auto'
                onClick={() => toggleExpanded(facet.key)}
              >
                {open ? 'Show less' : `+ ${hidden} more`}
              </Button>
            )}
          </Flex>
        );
      })}
    </Flex>
  );
};
