import { faHourglassHalf, faPause, faPlay } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Popconfirm, Popover, Tag, Typography, message } from 'antd';
import { ReactNode, useMemo, useState } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import type { AutoPromotionHold, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import {
  useGetStageAutoPromotionCandidates,
  useResumeStageAutoPromotion
} from '@ui/gen/api/v2/core/core';

import {
  autoPromotionHoldStateActive,
  autoPromotionHoldStatePending,
  getAutoPromotionCandidate,
  originKey,
  originLabel,
  type OriginLike
} from './auto-promotion';

type HoldEntry = {
  key: string;
  hold: AutoPromotionHold;
  origin?: OriginLike;
  focused: boolean;
};

type AutoPromotionHoldsPopoverProps = {
  stage: Stage;
  children?: ReactNode;
  focusOrigin?: OriginLike;
  placement?: 'bottom' | 'bottomLeft' | 'bottomRight' | 'top' | 'topLeft' | 'topRight';
};

const originFromKey = (key: string): OriginLike | undefined => {
  const [kind, name] = key.split('/');
  if (!kind || !name) {
    return undefined;
  }
  return { kind, name };
};

const holdStateLabel = (hold: AutoPromotionHold) =>
  hold.state === autoPromotionHoldStatePending ? 'Pending' : 'Paused';

const holdStateColor = (hold: AutoPromotionHold) =>
  hold.state === autoPromotionHoldStatePending ? 'gold' : 'volcano';

export const AutoPromotionHoldsPopover = ({
  stage,
  children,
  focusOrigin,
  placement = 'bottomRight'
}: AutoPromotionHoldsPopoverProps) => {
  const [open, setOpen] = useState(false);
  const [resumingOrigin, setResumingOrigin] = useState('');

  const projectName = stage?.metadata?.namespace || '';
  const stageName = stage?.metadata?.name || '';
  const focusOriginKey = originKey(focusOrigin);

  const entries = useMemo<HoldEntry[]>(() => {
    const holds = stage?.status?.autoPromotionHolds || {};
    return Object.entries(holds)
      .map(([key, hold]) => ({
        key,
        hold,
        origin: hold?.freight?.origin || originFromKey(key),
        focused: Boolean(focusOriginKey && focusOriginKey === key)
      }))
      .sort((lhs, rhs) => {
        if (lhs.focused !== rhs.focused) {
          return lhs.focused ? -1 : 1;
        }
        if (lhs.hold.state !== rhs.hold.state) {
          return lhs.hold.state === autoPromotionHoldStateActive ? -1 : 1;
        }
        return lhs.key.localeCompare(rhs.key);
      });
  }, [stage, focusOriginKey]);

  const activeCount = entries.filter(
    (entry) => entry.hold.state === autoPromotionHoldStateActive
  ).length;
  const pendingCount = entries.filter(
    (entry) => entry.hold.state === autoPromotionHoldStatePending
  ).length;

  const candidatesQuery = useGetStageAutoPromotionCandidates(projectName, stageName, {
    query: {
      enabled: Boolean(open && projectName && stageName && entries.length)
    }
  });
  const candidates =
    candidatesQuery.data?.status === 200 ? candidatesQuery.data.data.candidates : undefined;

  const resumeMutation = useResumeStageAutoPromotion({
    mutation: {
      onSuccess: () => {
        message.success('Auto-promotion resumed');
        setOpen(false);
      },
      onSettled: () => setResumingOrigin('')
    }
  });

  if (!entries.length) {
    return null;
  }

  const renderFreightLink = (freightName?: string) => {
    if (!freightName || !projectName) {
      return freightName || 'unknown Freight';
    }
    return (
      <Link
        to={generatePath(paths.freight, {
          name: projectName,
          freightName
        })}
        onClick={(event) => event.stopPropagation()}
        style={{ overflowWrap: 'anywhere' }}
      >
        {freightName}
      </Link>
    );
  };

  const resumeAutoPromotion = (entry: HoldEntry) => {
    if (!projectName || !stageName || !entry.origin?.kind || !entry.origin?.name) {
      return;
    }
    setResumingOrigin(entry.key);
    resumeMutation.mutate({
      project: projectName,
      stage: stageName,
      data: {
        origin: {
          kind: entry.origin.kind,
          name: entry.origin.name
        }
      }
    });
  };

  const triggerNode = children || (
    <Button
      aria-label='Auto-promotion holds'
      size='small'
      icon={<FontAwesomeIcon icon={activeCount > 0 ? faPause : faHourglassHalf} size='sm' />}
    />
  );

  const content = (
    <Flex vertical gap={10} className='min-w-[300px] max-w-[420px]'>
      <Typography.Text type='secondary' className='text-xs'>
        Auto-promotion is paused per Freight origin. Resuming one origin does not resume the others.
      </Typography.Text>

      {entries.map((entry) => {
        const isPending = entry.hold.state === autoPromotionHoldStatePending;
        const origin = entry.origin;
        const candidateName = getAutoPromotionCandidate(candidates, origin)?.freight?.name;
        const isResuming = resumingOrigin === entry.key && resumeMutation.isPending;

        const candidateDescription = candidatesQuery.isLoading ? (
          <>Checking the current auto-promotion candidate.</>
        ) : candidateName ? (
          <>Current auto-promotion candidate is {renderFreightLink(candidateName)}.</>
        ) : (
          <>No current auto-promotion candidate exists for this origin.</>
        );

        return (
          <Flex
            key={entry.key}
            align='flex-start'
            justify='space-between'
            gap={12}
            className='border-0 border-t border-solid border-gray-100 pt-2 first:border-t-0 first:pt-0'
          >
            <Flex vertical gap={3} className='min-w-0'>
              <Flex align='center' gap={6} wrap='wrap'>
                <Typography.Text strong className='text-xs'>
                  {originLabel(origin)}
                </Typography.Text>
                <Tag
                  bordered={false}
                  color={holdStateColor(entry.hold)}
                  className='m-0 text-[10px]'
                >
                  {holdStateLabel(entry.hold)}
                </Tag>
              </Flex>

              <Typography.Text type='secondary' className='text-xs'>
                {isPending ? 'Rollback Promotion is still settling' : 'Paused after rollback'} to{' '}
                {renderFreightLink(entry.hold.freight?.name)}.
              </Typography.Text>

              {entry.hold.reason && (
                <Typography.Text type='secondary' className='text-xs'>
                  Reason: {entry.hold.reason}
                </Typography.Text>
              )}

              <Typography.Text type='secondary' className='text-xs'>
                {candidateDescription}
              </Typography.Text>
            </Flex>

            {isPending ? (
              <Button size='small' disabled>
                Pending
              </Button>
            ) : (
              <Popconfirm
                title={`Resume auto-promotion for ${originLabel(origin)}?`}
                description={
                  <span style={{ display: 'block', maxWidth: 300, overflowWrap: 'anywhere' }}>
                    {candidateDescription}
                  </span>
                }
                okText='Resume'
                cancelText='Cancel'
                onConfirm={() => resumeAutoPromotion(entry)}
                disabled={candidatesQuery.isLoading || resumeMutation.isPending}
              >
                <Button
                  size='small'
                  icon={<FontAwesomeIcon icon={faPlay} />}
                  loading={isResuming}
                  disabled={candidatesQuery.isLoading || resumeMutation.isPending}
                >
                  Resume
                </Button>
              </Popconfirm>
            )}
          </Flex>
        );
      })}
    </Flex>
  );

  return (
    <Popover
      trigger='click'
      placement={placement}
      open={open}
      onOpenChange={setOpen}
      title={
        <Flex align='center' gap={8}>
          <FontAwesomeIcon icon={activeCount > 0 ? faPause : faHourglassHalf} />
          <span>
            {activeCount > 0 && `${activeCount} paused`}
            {activeCount > 0 && pendingCount > 0 && ', '}
            {pendingCount > 0 && `${pendingCount} pending`}
          </span>
        </Flex>
      }
      content={content}
    >
      <span
        className='inline-flex'
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        {triggerNode}
      </span>
    </Popover>
  );
};
