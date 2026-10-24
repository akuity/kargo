import { faBullseye, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { TablePaginationConfig } from 'antd/es/table/interface';
import { useState } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';

import { freightNames } from '../model/round-utils';
import { visibleLabels } from '../model/target-labels';
import { TargetRow } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { FreightLink } from '../shared/freight-link';
import { StateDot } from '../shared/state-visuals';

import { TargetDetails } from './target-details';

// A Target governed by many Stages would otherwise fill its row with links;
// past this many, the rest fold into a "+N more" tag that lists them on hover.
const maxStageLinks = 4;

// Likewise for labels, which a Target may carry any number of.
const maxLabelTags = 4;

/**
 * TargetTable lists Targets row by row, each opening on click to everything
 * about it that does not fit on one line. It is the one list of Targets on the
 * page, used both by the drawer a card opens and by the grouped list view, so
 * a Target reads the same wherever it is met.
 */
export const TargetTable = ({
  project,
  rows,
  groupKey,
  freightLabel,
  pagination = false
}: {
  project: string;
  rows: TargetRow[];
  // The label key the list is grouped by. Its tag says the same thing on every
  // row of a group, so the Labels column leaves it out.
  groupKey?: string;
  freightLabel: (name: string) => string;
  pagination?: TablePaginationConfig | false;
}) => {
  const [expanded, setExpanded] = useState<string[]>([]);

  const toggle = (key: string) =>
    setExpanded((current) =>
      current.includes(key) ? current.filter((open) => open !== key) : [...current, key]
    );

  const columns: ColumnsType<TargetRow> = [
    {
      title: 'Target',
      key: 'target',
      render: (_, row) => (
        <Typography.Text className='text-xs font-semibold'>
          <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
          {row.target.name}
        </Typography.Text>
      )
    },
    {
      title: 'Labels',
      key: 'labels',
      width: '28%',
      render: (_, row) => {
        const labels = visibleLabels(row.target).filter(([key]) => key !== groupKey);
        return (
          <Flex gap={4} wrap>
            {labels.slice(0, maxLabelTags).map(([key, value]) => (
              <Tag key={key} bordered={false} className='m-0 text-xs'>
                {key}={value}
              </Tag>
            ))}
            {labels.length > maxLabelTags && (
              <Tooltip
                title={labels
                  .slice(maxLabelTags)
                  .map(([key, value]) => `${key}=${value}`)
                  .join(', ')}
              >
                <Tag bordered={false} className='m-0 text-xs'>
                  +{labels.length - maxLabelTags}
                </Tag>
              </Tooltip>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Stages',
      key: 'stages',
      width: 160,
      render: (_, row) =>
        row.stages.length ? (
          <Flex gap={10} wrap align='center'>
            {row.stages.slice(0, maxStageLinks).map(({ stage }) => (
              <Link
                key={stage.metadata?.name}
                className='text-xs'
                to={generatePath(paths.stage, {
                  name: project,
                  stageName: stage.metadata?.name || ''
                })}
              >
                {stage.metadata?.name}
              </Link>
            ))}
            {row.stages.length > maxStageLinks && (
              <Tooltip
                title={row.stages
                  .slice(maxStageLinks)
                  .map(({ stage }) => stage.metadata?.name)
                  .join(', ')}
              >
                <Tag className='m-0 text-xs'>+{row.stages.length - maxStageLinks}</Tag>
              </Tooltip>
            )}
          </Flex>
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            none
          </Typography.Text>
        )
    },
    {
      title: 'Freight',
      key: 'freight',
      width: 200,
      render: (_, row) => {
        // What the Stages that have left this Target behind have moved on to.
        const ahead = [
          ...new Set(
            row.stages
              .filter(({ upToDate }) => upToDate === false)
              .flatMap(({ stage }) => freightNames(stage.status?.freightHistory?.[0]))
          )
        ];
        return (
          <Flex gap={6} wrap align='center'>
            {row.freight.length ? (
              row.freight.map((name) => (
                <FreightLink
                  key={name}
                  project={project}
                  name={name}
                  label={freightLabel(name)}
                  className='text-xs'
                />
              ))
            ) : (
              <Typography.Text type='secondary' className='text-xs'>
                none
              </Typography.Text>
            )}
            {ahead.length > 0 && (
              <Tooltip title={`The Stage has moved on to ${ahead.join(', ')}`}>
                <Tag color='warning' className='m-0 text-xs font-mono'>
                  behind {ahead.map(freightLabel).join(', ')}
                </Tag>
              </Tooltip>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Health',
      key: 'state',
      width: 150,
      render: (_, row) => (
        <Flex gap={6} align='center'>
          <StateDot state={row.state} />
          <span className='text-xs'>{targetStatePresentations[row.state].label}</span>
        </Flex>
      )
    }
  ];

  return (
    <Table
      dataSource={rows}
      columns={columns}
      size='small'
      pagination={pagination}
      rowKey={(row) => row.target.name || ''}
      expandable={{
        expandedRowKeys: expanded,
        onExpandedRowsChange: (keys) => setExpanded(keys.map(String)),
        expandedRowRender: (row) => (
          <TargetDetails project={project} row={row} freightLabel={freightLabel} />
        ),
        // A chevron that turns, rather than Ant's plus/minus box: the row opens
        // onto more of itself, it is not a thing being added. It stays a real
        // button so that it keeps its keyboard handling, and so that the row's
        // own click handler knows to leave it alone.
        expandIcon: ({ expanded: open, onExpand, record }) => (
          <button
            type='button'
            aria-expanded={open}
            aria-label={open ? 'Hide details' : 'Show details'}
            className='border-0 bg-transparent p-0 mr-1 cursor-pointer text-gray-400'
            onClick={(event) => onExpand(record, event)}
          >
            <FontAwesomeIcon
              icon={faChevronRight}
              size='xs'
              className='transition-transform duration-200'
              style={{ transform: open ? 'rotate(90deg)' : undefined }}
            />
          </button>
        )
      }}
      onRow={(row) => ({
        className: 'cursor-pointer',
        onClick: (event) => {
          // Links and the expand chevron in a row do their own thing; only the
          // row's own surface toggles it.
          if ((event.target as HTMLElement).closest('a, button')) {
            return;
          }
          // A click that ends a text selection was a drag to copy a Target
          // name, not a click on the row.
          if (window.getSelection()?.toString()) {
            return;
          }
          toggle(row.target.name || '');
        }
      })}
    />
  );
};
