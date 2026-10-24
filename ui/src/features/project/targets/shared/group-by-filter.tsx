import { faEllipsisVertical } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Flex, Tag, Typography, theme } from 'antd';
import { useMemo } from 'react';

import { NO_GROUPING, STAGE_GROUPING, splitKeys } from '../model/target-grouping';

/**
 * GroupByFilter chooses the label key a fleet is sliced along, as a row of
 * pills: Stage first, when there are Stages to group by, then the keys the
 * Targets actually carry, none of them picked to begin with. Stage never folds
 * into the dropdown: it is the one axis every project with a fleet has. Each pill toggles, so the way back to an ungrouped fleet is to unpick
 * the key in force rather than to find a "None" among the axes -- not grouping
 * is the absence of a choice, not a choice of its own.
 *
 * Keys that do not fit fold into a dropdown behind a dots pill, and that pill
 * takes the selected background when the axis in force is one of them -- so the
 * row always shows whether what is selected is on it or behind the dots,
 * without the dots themselves changing shape.
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
  const { inline, overflow } = useMemo(() => splitKeys(keys), [keys]);
  const overflowSelected = overflow.includes(value);

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
      {inline.map((key) => (
        <Tag.CheckableTag
          key={key}
          className='m-0'
          style={outline(value === key)}
          disabled={disabled}
          checked={value === key}
          onChange={() => onChange(toggle(key))}
        >
          {key}
        </Tag.CheckableTag>
      ))}
      {overflow.length > 0 && (
        <Dropdown
          trigger={['click']}
          disabled={disabled}
          menu={{
            selectable: true,
            selectedKeys: overflowSelected ? [value] : [],
            items: overflow.map((key) => ({ key, label: key })),
            onClick: ({ key }) => onChange(toggle(key))
          }}
        >
          {/* The pill is the Dropdown's own child, so that the click handler
              Dropdown injects lands on something that forwards it. */}
          <Tag.CheckableTag
            className='m-0'
            style={outline(overflowSelected)}
            disabled={disabled}
            checked={overflowSelected}
            aria-label={`${overflow.length} more grouping axes`}
            // The pill opens the dropdown; it is not a choice of its own.
            onChange={() => undefined}
          >
            <FontAwesomeIcon icon={faEllipsisVertical} />
          </Tag.CheckableTag>
        </Dropdown>
      )}
    </Flex>
  );
};
