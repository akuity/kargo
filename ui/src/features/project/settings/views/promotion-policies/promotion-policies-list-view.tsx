import { faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table, Tag, Tooltip, Typography } from 'antd';

import { selectorLines } from '@ui/features/common/selector/selector-utils';
import { PromotionPolicy } from '@ui/gen/api/v2/models';

import { isLegacyPolicy } from './promotion-policy-form-utils';

const StageCell = ({ promotionPolicy }: { promotionPolicy: PromotionPolicy }) => {
  if (isLegacyPolicy(promotionPolicy)) {
    return (
      <Flex align='center' gap={8}>
        <Typography.Text strong>{promotionPolicy.stage}</Typography.Text>
        <Tooltip title='Set through the deprecated stage field instead of a selector'>
          <Tag className='m-0'>Deprecated</Tag>
        </Tooltip>
      </Flex>
    );
  }

  const lines = selectorLines(promotionPolicy.stageSelector);

  if (!lines.length) {
    return <Typography.Text type='secondary'>Every Stage</Typography.Text>;
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

const AutoRollbackCell = ({ promotionPolicy }: { promotionPolicy: PromotionPolicy }) => {
  const { autoRollback } = promotionPolicy;

  if (!autoRollback) {
    return <Typography.Text type='secondary'>Off</Typography.Text>;
  }

  // the API treats an absent or empty list of verification phases as [Failed]
  const onVerification = autoRollback.onVerification?.length
    ? autoRollback.onVerification
    : ['Failed'];

  return (
    <Flex vertical gap={2}>
      {(
        [
          ['Verification', onVerification],
          ['Promotion', autoRollback.onPromotion ?? []]
        ] as const
      ).map(([label, phases]) => (
        <Flex key={label} gap={8} align='baseline'>
          <Typography.Text type='secondary' className='w-20 shrink-0 text-xs'>
            {label}
          </Typography.Text>
          <Typography.Text className='text-xs' type={phases.length ? undefined : 'secondary'}>
            {phases.length ? phases.join(', ') : '-'}
          </Typography.Text>
        </Flex>
      ))}
    </Flex>
  );
};

/**
 * Policies have no name of their own -- their position in the full list is
 * their identity, so each row carries it. The index AntD hands to a column's
 * render counts from the top of the current page, so it can't stand in.
 */
type PromotionPolicyRow = { promotionPolicy: PromotionPolicy; index: number };

type PromotionPoliciesListViewProps = {
  promotionPolicies: PromotionPolicy[];
  loading?: boolean;
  showAutoRollback: boolean;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
};

export const PromotionPoliciesListView = ({
  promotionPolicies,
  loading,
  showAutoRollback,
  onEdit,
  onDelete
}: PromotionPoliciesListViewProps) => (
  <Table<PromotionPolicyRow>
    loading={loading}
    rowKey='index'
    dataSource={promotionPolicies.map((promotionPolicy, index) => ({ promotionPolicy, index }))}
    pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
    size='small'
    scroll={{ x: 'max-content' }}
    locale={{ emptyText: 'No policies yet - promotions happen only when someone asks for them.' }}
    columns={[
      {
        key: 'stageSelector',
        title: 'Stages',
        render: (_, { promotionPolicy }) => <StageCell promotionPolicy={promotionPolicy} />
      },
      {
        key: 'autoPromotionEnabled',
        title: 'Auto-promotion',
        render: (_, { promotionPolicy }) => (
          <Tag color={promotionPolicy.autoPromotionEnabled ? 'green' : undefined} className='m-0'>
            {promotionPolicy.autoPromotionEnabled ? 'Enabled' : 'Disabled'}
          </Tag>
        )
      },
      ...(showAutoRollback
        ? [
            {
              key: 'autoRollback',
              title: 'Auto-rollback',
              render: (_: unknown, { promotionPolicy }: PromotionPolicyRow) => (
                <AutoRollbackCell promotionPolicy={promotionPolicy} />
              )
            }
          ]
        : []),
      {
        key: 'actions',
        render: (_, { promotionPolicy, index }) => (
          <Flex gap={8} justify='end'>
            {/* stage and stageSelector are mutually exclusive, so the form can't
                edit a policy on the deprecated field without rewriting it */}
            <Tooltip
              title={
                isLegacyPolicy(promotionPolicy)
                  ? 'Set through the deprecated stage field, so it cannot be edited here. Change it in the ProjectConfig YAML, or delete it and create a replacement that uses a selector.'
                  : undefined
              }
            >
              <Button
                icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                onClick={() => onEdit(index)}
                disabled={isLegacyPolicy(promotionPolicy)}
                color='default'
                variant='filled'
                size='small'
              >
                Edit
              </Button>
            </Tooltip>
            <Button
              icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
              onClick={() => onDelete(index)}
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
