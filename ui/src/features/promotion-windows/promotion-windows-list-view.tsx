import { faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table, Tag, Typography } from 'antd';
import classNames from 'classnames';
import { format } from 'date-fns';
import { RRule } from 'rrule';

import {
  PromotionPolicySelector,
  PromotionWindow,
  PromotionWindowKind
} from '@ui/gen/api/v2/models';

import { occurrenceColors } from './occurrence-colors';
import { dtstartLiteral } from './parse-promotion-windows';
import { selectorLines } from './promotion-window-selector-utils';
import { fromViewerClockDate } from './viewer-clock';

const SelectorCell = ({ selector }: { selector?: PromotionPolicySelector }) => {
  const lines = selectorLines(selector);

  if (!lines.length) {
    return <Typography.Text type='secondary'>-</Typography.Text>;
  }

  return (
    <Flex vertical gap={2}>
      {lines.map((line, index) => (
        <Typography.Text key={index} className='text-xs'>
          {line}
        </Typography.Text>
      ))}
    </Flex>
  );
};

type PromotionWindowsListViewProps = {
  scope: 'project' | 'cluster';
  promotionWindows: PromotionWindow[];
  onEdit: (promotionWindow: PromotionWindow) => void;
  onDelete: (promotionWindow: PromotionWindow) => void;
};

export const PromotionWindowsListView = ({
  scope,
  promotionWindows,
  onEdit,
  onDelete
}: PromotionWindowsListViewProps) => (
  <Table<PromotionWindow>
    rowKey={(promotionWindow) => promotionWindow.name}
    dataSource={promotionWindows}
    pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
    size='small'
    scroll={{ x: 'max-content' }}
    locale={{ emptyText: 'No windows yet - promotions are unconstrained.' }}
    columns={[
      {
        key: 'name',
        title: 'Name',
        render: (_, promotionWindow) => (
          <Flex vertical gap={2}>
            <Flex align='center' gap={8}>
              <span
                className={classNames(
                  'size-2 shrink-0 rounded-full',
                  occurrenceColors(promotionWindow.kind).dot
                )}
              />
              <Typography.Text strong delete={promotionWindow.disabled}>
                {promotionWindow.name}
              </Typography.Text>
              {promotionWindow.disabled && <Tag className='m-0'>Disabled</Tag>}
            </Flex>
            {promotionWindow.description && (
              <Typography.Text type='secondary' className='text-xs'>
                {promotionWindow.description}
              </Typography.Text>
            )}
          </Flex>
        )
      },
      {
        key: 'kind',
        title: 'Kind',
        render: (_, promotionWindow) => (
          <Tag
            color={
              promotionWindow.kind === PromotionWindowKind.PromotionWindowKindAllow
                ? 'green'
                : 'red'
            }
            className='m-0'
          >
            {promotionWindow.kind}
          </Tag>
        )
      },
      {
        key: 'dtstartDtend',
        title: 'Start / End',
        render: (_, promotionWindow) => (
          <Flex vertical gap={2}>
            {(
              [
                ['Start', promotionWindow.dtstart],
                ['End', promotionWindow.dtend]
              ] as const
            ).map(([label, literal]) => {
              const options = literal ? RRule.fromString(dtstartLiteral(literal)).options : null;

              return (
                <Flex key={label} gap={8} align='baseline'>
                  <Typography.Text type='secondary' className='w-10 shrink-0 text-xs'>
                    {label}
                  </Typography.Text>
                  <Typography.Text className='text-xs' type={options ? undefined : 'secondary'}>
                    {options
                      ? `${format(fromViewerClockDate(options.dtstart), 'MMM d, yyyy HH:mm')}${
                          options.tzid ? ` (${options.tzid})` : ''
                        }`
                      : '-'}
                  </Typography.Text>
                </Flex>
              );
            })}
          </Flex>
        )
      },
      {
        key: 'rrule',
        title: 'Recurrence',
        render: (_, promotionWindow) => (
          <Typography.Text className='text-xs'>
            {promotionWindow.rrule
              ? RRule.fromString(
                  `${dtstartLiteral(promotionWindow.dtstart)}\nRRULE:${promotionWindow.rrule}`
                ).toText()
              : 'Does not repeat'}
          </Typography.Text>
        )
      },
      {
        key: 'stageSelector',
        title: 'Stage Selector',
        render: (_, promotionWindow) => <SelectorCell selector={promotionWindow.stageSelector} />
      },
      ...(scope === 'cluster'
        ? [
            {
              key: 'projectSelector',
              title: 'Project Selector',
              render: (_: unknown, promotionWindow: PromotionWindow) => (
                <SelectorCell selector={promotionWindow.projectSelector} />
              )
            }
          ]
        : []),
      {
        key: 'actions',
        render: (_, promotionWindow) => (
          <Flex gap={8} justify='end'>
            <Button
              icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
              onClick={() => onEdit(promotionWindow)}
              color='default'
              variant='filled'
              size='small'
            >
              Edit
            </Button>
            <Button
              icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
              onClick={() => onDelete(promotionWindow)}
              color='danger'
              variant='filled'
              size='small'
            >
              Delete
            </Button>
          </Flex>
        )
      }
    ]}
  />
);
