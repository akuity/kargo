import { faHourglassHalf, faPause, faPlay } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Button, Drawer, Flex, Radio, Tag, Typography, message } from 'antd';
import { ReactNode, useEffect, useMemo, useState } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import type { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import {
  useGetStageAutoPromotionCandidates,
  useResumeStageAutoPromotion
} from '@ui/gen/api/v2/core/core';

import {
  AutoPromotionHoldEntry,
  autoPromotionHoldStateActive,
  autoPromotionHoldStatePending,
  getAutoPromotionCandidate,
  getAutoPromotionHoldEntries,
  originLabel,
  type OriginLike
} from './auto-promotion';

type ResumeAutoPromotionDrawerProps = {
  stage: Stage;
  open: boolean;
  onClose: () => void;
  focusOrigin?: OriginLike;
};

const holdStateLabel = (entry: AutoPromotionHoldEntry) =>
  entry.hold.state === autoPromotionHoldStatePending ? 'Pending' : 'Paused';

const holdStateColor = (entry: AutoPromotionHoldEntry) =>
  entry.hold.state === autoPromotionHoldStatePending ? 'gold' : 'volcano';

export const ResumeAutoPromotionDrawer = ({
  stage,
  open,
  onClose,
  focusOrigin
}: ResumeAutoPromotionDrawerProps) => {
  const [selectedOriginKey, setSelectedOriginKey] = useState('');

  const projectName = stage?.metadata?.namespace || '';
  const stageName = stage?.metadata?.name || '';

  const entries = useMemo(
    () => getAutoPromotionHoldEntries(stage, focusOrigin),
    [stage, focusOrigin]
  );
  const activeEntries = entries.filter(
    (entry) => entry.hold.state === autoPromotionHoldStateActive
  );
  const pendingEntries = entries.filter(
    (entry) => entry.hold.state === autoPromotionHoldStatePending
  );
  const activeEntryKeys = activeEntries.map((entry) => entry.key).join('|');

  useEffect(() => {
    if (!open) {
      return;
    }
    if (selectedOriginKey && activeEntries.some((entry) => entry.key === selectedOriginKey)) {
      return;
    }
    setSelectedOriginKey(activeEntries[0]?.key || '');
  }, [open, activeEntryKeys, selectedOriginKey]);

  const selectedEntry = activeEntries.find((entry) => entry.key === selectedOriginKey);

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
        onClose();
      }
    }
  });

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
        onClick={onClose}
        style={{ overflowWrap: 'anywhere' }}
      >
        {freightName}
      </Link>
    );
  };

  const renderCandidate = (entry?: AutoPromotionHoldEntry): ReactNode => {
    if (!entry) {
      return 'No held origin selected.';
    }
    if (candidatesQuery.isLoading) {
      return 'Checking the current auto-promotion candidate.';
    }

    const candidateName = getAutoPromotionCandidate(candidates, entry.origin)?.freight?.name;
    if (!candidateName) {
      return 'No current auto-promotion candidate exists for this origin.';
    }

    return <>Current auto-promotion candidate is {renderFreightLink(candidateName)}.</>;
  };

  const resumeAutoPromotion = () => {
    if (!projectName || !stageName || !selectedEntry?.origin?.kind || !selectedEntry.origin.name) {
      return;
    }

    resumeMutation.mutate({
      project: projectName,
      stage: stageName,
      data: {
        origin: {
          kind: selectedEntry.origin.kind,
          name: selectedEntry.origin.name
        }
      }
    });
  };

  const renderHoldSummary = (entry: AutoPromotionHoldEntry) => (
    <Flex
      key={entry.key}
      vertical
      gap={6}
      className='rounded border border-solid border-gray-200 p-3'
    >
      <Flex align='center' gap={8} wrap='wrap'>
        <Typography.Text strong>{originLabel(entry.origin)}</Typography.Text>
        <Tag bordered={false} color={holdStateColor(entry)} className='m-0'>
          <FontAwesomeIcon
            icon={entry.hold.state === autoPromotionHoldStatePending ? faHourglassHalf : faPause}
            className='mr-1'
          />
          {holdStateLabel(entry)}
        </Tag>
      </Flex>

      <Typography.Text type='secondary'>
        {entry.hold.state === autoPromotionHoldStatePending
          ? 'Rollback Promotion is still settling'
          : 'Paused after rollback'}{' '}
        to {renderFreightLink(entry.hold.freight?.name)}.
      </Typography.Text>

      {entry.hold.reason && (
        <Typography.Text type='secondary'>Reason: {entry.hold.reason}</Typography.Text>
      )}

      <Typography.Text type='secondary'>{renderCandidate(entry)}</Typography.Text>
    </Flex>
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={720}
      title={
        <Flex align='center' gap={8}>
          <FontAwesomeIcon icon={faPlay} />
          Resume auto-promotion
        </Flex>
      }
      footer={
        <Flex justify='space-between' gap={12}>
          <Button onClick={onClose}>Cancel</Button>
          <Button
            type='primary'
            icon={<FontAwesomeIcon icon={faPlay} />}
            onClick={resumeAutoPromotion}
            loading={resumeMutation.isPending}
            disabled={!selectedEntry || candidatesQuery.isLoading}
          >
            Resume auto-promotion
          </Button>
        </Flex>
      }
    >
      <Flex vertical gap={16}>
        <Alert
          showIcon
          type='info'
          message='Resuming clears the active hold for one Freight origin.'
          description='Kargo will evaluate auto-promotion normally after the hold is cleared. Resume does not directly create a Promotion.'
        />

        {activeEntries.length === 0 && (
          <Alert
            showIcon
            type='warning'
            message='No active auto-promotion hold can be resumed.'
            description='A pending hold is still waiting for its rollback Promotion to settle.'
          />
        )}

        {activeEntries.length === 1 && selectedEntry && renderHoldSummary(selectedEntry)}

        {activeEntries.length > 1 && (
          <Radio.Group
            className='w-full'
            value={selectedOriginKey}
            onChange={(event) => setSelectedOriginKey(event.target.value)}
          >
            <Flex vertical gap={10}>
              {activeEntries.map((entry) => (
                <Radio key={entry.key} value={entry.key} className='w-full m-0'>
                  <div className='ml-2'>{renderHoldSummary(entry)}</div>
                </Radio>
              ))}
            </Flex>
          </Radio.Group>
        )}

        {pendingEntries.length > 0 && activeEntries.length > 0 && (
          <Flex vertical gap={8}>
            <Typography.Text strong className='text-xs uppercase'>
              Pending holds
            </Typography.Text>
            {pendingEntries.map(renderHoldSummary)}
          </Flex>
        )}
      </Flex>
    </Drawer>
  );
};
