import { faChevronDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Flex, Tag, Typography, theme } from 'antd';
import { useState } from 'react';

import { NO_GROUPING, STAGE_GROUPING } from '../model/target-grouping';

/**
 * GroupByFilter chooses the axis a fleet is sliced along, as two pills: Stage,
 * when there are Stages to group by, and Label, a dropdown of every label key
 * the Targets actually carry. Neither is picked to begin with. Each choice
 * toggles, so the way back to an ungrouped fleet is to unpick the axis in force
 * rather than to find a "None" among the axes -- not grouping is the absence of
 * a choice, not a choice of its own.
 *
 * The Label pill takes the selected background and names the key while a label
 * is the axis in force, so the row always shows what the fleet is grouped by
 * without opening the dropdown.
 */
export const GroupByFilter = ({
  keys,
  byStage,
  value,
  onChange,
  disabled
}: {
  // The label keys available as axes, in the order they should be offered.
  keys: string[];
  // Whether grouping by Stage is offered.
  byStage?: boolean;
  // The key in force, STAGE_GROUPING, or NO_GROUPING.
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) => {
  const { token } = theme.useToken();
  const labelSelected = keys.includes(value);
  const [labelMenuOpen, setLabelMenuOpen] = useState(false);

  // Picking the key already in force unpicks it, leaving the fleet ungrouped.
  const toggle = (key: string) => (value === key ? NO_GROUPING : key);

  // An unpicked pill is a filled shape with no edge of its own, which reads as
  // a label rather than something to press. Outlining it gives it one. The
  // picked pill carries the same border in its own colour, so toggling does not
  // shift the row by a pixel.
  const outline = (picked: boolean) => ({
    border: `1px solid ${picked ? 'transparent' : token.colorBorder}`
  });

  return (
    <Flex gap={8} align='center' wrap>
      <Typography.Text type='secondary' className='text-xs shrink-0'>
        Group by
      </Typography.Text>
      {byStage && (
        <Tag.CheckableTag
          className='m-0'
          style={outline(value === STAGE_GROUPING)}
          disabled={disabled}
          checked={value === STAGE_GROUPING}
          onChange={() => onChange(toggle(STAGE_GROUPING))}
        >
          Stage
        </Tag.CheckableTag>
      )}
      {keys.length > 0 && (
        <Dropdown
          trigger={['click']}
          disabled={disabled}
          open={labelMenuOpen}
          onOpenChange={setLabelMenuOpen}
          menu={{
            selectable: true,
            selectedKeys: labelSelected ? [value] : [],
            items: keys.map((key) => ({ key, label: key })),
            onClick: ({ key }) => onChange(toggle(key))
          }}
        >
          {/* The pill is the Dropdown's own child, so that the click handler
              Dropdown injects lands on something that forwards it. */}
          <Tag.CheckableTag
            className='m-0 max-w-full'
            style={outline(labelSelected)}
            disabled={disabled}
            checked={labelSelected}
            aria-haspopup='menu'
            aria-expanded={labelMenuOpen}
            // The pill opens the dropdown; it is not a choice of its own.
            onChange={() => undefined}
            // Dropdown only listens for clicks, and the pill swallows Space
            // without forwarding it, so the keyboard opens the menu here. Once
            // open, Tab moves focus into the menu.
            onKeyDown={(e) => {
              if (disabled || (e.key !== 'Enter' && e.key !== ' ')) {
                return;
              }
              e.preventDefault();
              setLabelMenuOpen(true);
            }}
          >
            <Flex gap={6} align='center' className='min-w-0'>
              {/* Label keys can run long; cap the pill so the chevron stays
                  on screen, and keep the full key on hover. */}
              <span className='truncate max-w-[200px]' title={labelSelected ? value : undefined}>
                {labelSelected ? value : 'Label'}
              </span>
              <FontAwesomeIcon icon={faChevronDown} className='text-[10px] shrink-0' />
            </Flex>
          </Tag.CheckableTag>
        </Dropdown>
      )}
    </Flex>
  );
};
