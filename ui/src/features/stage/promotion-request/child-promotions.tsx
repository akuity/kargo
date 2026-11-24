import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Input, Table, Tag, Tooltip, Typography, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import { useMemo, useState } from 'react';
import { Link, generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PromotionStepStatus } from '@ui/features/common/promotion-directive-step-status/promotion-step-status';
import { PhaseFilterChips } from '@ui/features/common/promotion-status/phase-filter-chips';
import {
  PromotionPhase,
  getPromotionPhasePresentation
} from '@ui/features/common/promotion-status/promotion-phase';
import { Promotion } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { ChildPromotion, childPhaseFilters, matchesChild, stepStatuses } from './round-detail';

const pageSizeOptions = [10, 20, 50, 100];

// Below this a segment stops reading as a segment, so it is the floor the bar
// shrinks to rather than letting a long promotion smear into a dotted line.
const minStepWidth = 2;

// How many segments the column fits at that floor. Gaps do not shrink, so past
// this they would outweigh the bars. The rest are cut and counted; the tooltip
// still names every step.
const maxVisibleSteps = 24;

// StepBar draws one segment per step of a child Promotion, so a reader can see
// how far into its steps it got without opening it. A step with no status
// never ran, and is drawn as an empty track rather than an outcome.
const StepBar = ({ promotion }: { promotion?: Promotion }) => {
  const { token } = theme.useToken();
  const statuses = stepStatuses(promotion);

  if (!statuses.length) {
    return null;
  }

  const colorFor = (status?: string) => {
    switch (status) {
      case PromotionStepStatus.SUCCEEDED:
        return token.colorSuccess;
      case PromotionStepStatus.FAILED:
      case PromotionStepStatus.ERRORED:
      case PromotionStepStatus.ABORTED:
        return token.colorError;
      case PromotionStepStatus.RUNNING:
        return token.colorInfo;
      case PromotionStepStatus.SKIPPED:
        return token.colorTextQuaternary;
      default:
        return token.colorFillSecondary;
    }
  };

  const description = statuses
    .map((status, index) => `${index + 1}. ${status ?? 'not run'}`)
    .join(', ');

  const shown = statuses.slice(0, maxVisibleSteps);
  const cut = statuses.length - shown.length;

  return (
    <Flex gap={2} align='center' title={description} role='img' aria-label={description}>
      {shown.map((status, index) => (
        <div
          key={index}
          style={{
            flex: '0 1 14px',
            minWidth: minStepWidth,
            height: 5,
            borderRadius: 2,
            background: colorFor(status)
          }}
        />
      ))}
      {cut > 0 && (
        <span className='text-xs whitespace-nowrap' style={{ color: token.colorTextSecondary }}>
          +{cut}
        </span>
      )}
    </Flex>
  );
};

/**
 * ChildPromotions lists the Promotions one round fanned out into: one per
 * Target, what became of it, how far through its steps it got, and what it
 * said if it stopped.
 *
 * The chips and the search box are the only way to narrow it, because this is
 * where a reader lands from "6 of 136 did not succeed" and the first thing
 * they want is those six, by phase or by the label the trouble clusters on.
 */
export const ChildPromotions = ({
  project,
  rows,
  loading
}: {
  project: string;
  rows: ChildPromotion[];
  loading: boolean;
}) => {
  const navigate = useNavigate();
  const [phase, setPhase] = useState<PromotionPhase | undefined>();
  const [search, setSearch] = useState('');

  const pathOf = (row: ChildPromotion) =>
    row.promotion?.metadata?.name
      ? generatePath(paths.promotion, {
          name: project,
          promotionId: row.promotion.metadata.name
        })
      : undefined;

  const shown = useMemo(
    () => rows.filter((row) => (!phase || row.phase === phase) && matchesChild(row, search)),
    [rows, phase, search]
  );

  const columns: ColumnsType<ChildPromotion> = [
    {
      title: 'Phase',
      width: 120,
      render: (_, row) => {
        const { icon, tagColor, spin } = getPromotionPhasePresentation(row.phase);
        return (
          <Tag className='m-0' color={tagColor} icon={<FontAwesomeIcon icon={icon} spin={spin} />}>
            {row.phase}
          </Tag>
        );
      }
    },
    {
      title: 'Target',
      render: (_, row) => {
        const name = row.promotion?.metadata?.name;
        return (
          <Flex vertical gap={4} className='min-w-0'>
            {name ? (
              <Tooltip title={`Promotion ${name}`}>
                <Link className='text-xs font-mono' to={pathOf(row) || ''}>
                  {row.target}
                </Link>
              </Tooltip>
            ) : (
              <Tooltip title='No Promotion has been created for this Target yet'>
                <Typography.Text className='text-xs font-mono'>{row.target}</Typography.Text>
              </Tooltip>
            )}
            <Flex gap={4} wrap>
              {Object.entries(row.labels).map(([key, value]) => (
                <Tag key={key} className='m-0 text-xs'>
                  {key}={value}
                </Tag>
              ))}
            </Flex>
          </Flex>
        );
      }
    },
    {
      title: 'Steps',
      width: 110,
      render: (_, row) => <StepBar promotion={row.promotion} />
    },
    {
      title: 'Message',
      render: (_, row) => {
        const message = row.promotion?.status?.message;
        if (!message) {
          return null;
        }
        const failed = row.phase === 'Failed' || row.phase === 'Errored';
        return (
          <Typography.Text type={failed ? 'danger' : 'secondary'} className='text-xs font-mono'>
            {message}
          </Typography.Text>
        );
      }
    },
    {
      title: 'Finished',
      width: 160,
      render: (_, row) => {
        const finished = parseDate(row.promotion?.status?.finishedAt);
        return finished ? (
          <Typography.Text className='text-xs'>
            {format(finished, 'MMM d, HH:mm:ss')}
          </Typography.Text>
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            -
          </Typography.Text>
        );
      }
    }
  ];

  return (
    <Flex vertical gap={12}>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Flex gap={12} align='center' wrap>
          <Typography.Title level={5} className='m-0'>
            Promotions
          </Typography.Title>
          <PhaseFilterChips
            filters={childPhaseFilters(rows)}
            selected={phase}
            onSelect={setPhase}
          />
        </Flex>
        <Input
          allowClear
          size='small'
          className='max-w-60'
          placeholder='Target or label'
          prefix={<FontAwesomeIcon icon={faMagnifyingGlass} className='text-gray-400' />}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </Flex>

      <Table
        dataSource={shown}
        columns={columns}
        size='small'
        loading={loading}
        rowKey={(row) => row.target}
        onRow={(row) => {
          const path = pathOf(row);
          // A Target whose Promotion has not been created yet has nothing to
          // open, so its row stays inert rather than offering a dead click.
          if (!path) {
            return {};
          }
          return {
            className: 'cursor-pointer',
            onClick: (event) => {
              if ((event.target as HTMLElement).closest('a')) {
                return;
              }
              navigate(path);
            }
          };
        }}
        pagination={{
          pageSize: pageSizeOptions[0],
          pageSizeOptions,
          showSizeChanger: true,
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} Promotions`
        }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                rows.length ? 'No Promotion matches this filter' : 'This round named no Targets'
              }
            />
          )
        }}
      />
    </Flex>
  );
};
