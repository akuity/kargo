import { Flex, Typography, theme } from 'antd';

import { SmallLabel } from '@ui/features/common/small-label';

import { ChildPromotion, FailureTally, failuresByLabel, failuresByStep } from './round-detail';

const TallyRow = ({ tally }: { tally: FailureTally }) => {
  const { token } = theme.useToken();
  return (
    <Flex justify='space-between' align='center' gap={12}>
      <Typography.Text className='text-xs font-mono min-w-0 truncate'>{tally.key}</Typography.Text>
      <span className='text-xs whitespace-nowrap'>
        <span style={{ color: token.colorError }}>{tally.failed}</span>
        {tally.total !== undefined && (
          <Typography.Text type='secondary'> / {tally.total}</Typography.Text>
        )}
      </span>
    </Flex>
  );
};

/**
 * WhereItFailed answers the question a reader opens a failed round with: is
 * this one broken step, or one broken slice of the fleet?
 *
 * By step says whether the failures share a cause. By label says whether they
 * share a place -- a tier, a region, a cloud -- counted against how many
 * Targets carry that label, so five failures out of five reads differently
 * from five out of sixty-eight.
 */
export const WhereItFailed = ({ rows }: { rows: ChildPromotion[] }) => {
  const byStep = failuresByStep(rows);
  const byLabel = failuresByLabel(rows);

  if (!byStep.length && !byLabel.length) {
    return null;
  }

  return (
    <Flex
      vertical
      gap={12}
      className='rounded-md p-3 border border-solid bg-white dark:bg-neutral-800 border-gray-200 dark:border-neutral-700'
    >
      <Typography.Text strong className='text-sm'>
        Where it failed
      </Typography.Text>

      {byStep.length > 0 && (
        <Flex vertical gap={6}>
          <SmallLabel>By step</SmallLabel>
          {byStep.map((tally) => (
            <TallyRow key={tally.key} tally={tally} />
          ))}
        </Flex>
      )}

      {byLabel.length > 0 && (
        <Flex vertical gap={6}>
          <SmallLabel>By label</SmallLabel>
          {byLabel.map((tally) => (
            <TallyRow key={tally.key} tally={tally} />
          ))}
        </Flex>
      )}
    </Flex>
  );
};
