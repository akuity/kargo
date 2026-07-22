import { faPlay } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Button, Drawer, Flex, Radio, Typography, message } from 'antd';
import { ReactNode, useMemo, useState } from 'react';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { usePromoteToStage } from '@ui/gen/api/v2/core/core';
import type { Stage } from '@ui/gen/api/v2/models';

import { AutoPromotionHoldEntry, getAutoPromotionHoldEntries, originLabel } from './auto-promotion';

type ResumeAutoPromotionDrawerProps = {
  stage: Stage;
  open: boolean;
  onClose: () => void;
};

export const ResumeAutoPromotionDrawer = ({
  stage,
  open,
  onClose
}: ResumeAutoPromotionDrawerProps) => {
  const navigate = useNavigate();
  const [selectedOriginKey, setSelectedOriginKey] = useState('');

  const projectName = stage?.metadata?.namespace || '';
  const stageName = stage?.metadata?.name || '';

  const entries = useMemo(() => getAutoPromotionHoldEntries(stage), [stage]);

  // selectedOriginKey holds only explicit user choices; the effective
  // selection falls back to the first held origin whenever the stored choice
  // no longer matches an existing hold.
  const effectiveKey = entries.some((entry) => entry.key === selectedOriginKey)
    ? selectedOriginKey
    : (entries[0]?.key ?? '');
  const selectedEntry = entries.find((entry) => entry.key === effectiveKey);

  const promoteMutation = usePromoteToStage({
    mutation: {
      onSuccess: (response) => {
        message.success('Auto-promotion resumed');
        onClose();
        const promotionName = response.data?.metadata?.name;
        if (promotionName) {
          navigate(
            generatePath(paths.promotion, {
              name: projectName,
              promotionId: promotionName
            })
          );
        }
      }
    }
  });

  const renderFreightLink = (freightName?: string): ReactNode => {
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

  const renderHoldSummary = (entry: AutoPromotionHoldEntry): ReactNode => (
    <Flex vertical gap={2}>
      <Typography.Text strong>{originLabel(entry.origin)}</Typography.Text>
      <Typography.Text type='secondary'>
        Held at {renderFreightLink(entry.hold.freightName)}.
      </Typography.Text>
      {entry.hold.actor && (
        <Typography.Text type='secondary'>Held by {entry.hold.actor}.</Typography.Text>
      )}
    </Flex>
  );

  const resumeAutoPromotion = () => {
    if (!projectName || !stageName || !effectiveKey) {
      return;
    }
    promoteMutation.mutate({
      project: projectName,
      stage: stageName,
      data: { origin: effectiveKey }
    });
  };

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
        <Flex justify='flex-end'>
          <Button
            type='primary'
            icon={<FontAwesomeIcon icon={faPlay} />}
            onClick={resumeAutoPromotion}
            loading={promoteMutation.isPending}
            disabled={!selectedEntry}
          >
            Resume auto-promotion
          </Button>
        </Flex>
      }
    >
      <div className='-mt-6 -mx-6'>
        <Alert
          banner
          type='info'
          message='Resuming promotes the Freight that auto-promotion would select for this origin.'
          description='Kargo creates a Promotion immediately and resumes auto-promotion for this origin.'
        />

        {entries.length === 0 && (
          <Alert
            banner
            type='warning'
            message='No auto-promotion hold can be resumed.'
            description='There are no auto-promotion holds on this Stage.'
          />
        )}
      </div>
      <Flex vertical gap={16} className='mt-4'>
        {entries.length === 1 && selectedEntry && renderHoldSummary(selectedEntry)}

        {entries.length > 1 && (
          <Radio.Group
            className='w-full'
            value={effectiveKey}
            onChange={(event) => setSelectedOriginKey(event.target.value)}
          >
            <Flex vertical gap={10}>
              {entries.map((entry) => (
                <Radio key={entry.key} value={entry.key} className='w-full m-0'>
                  <div className='ml-2'>{renderHoldSummary(entry)}</div>
                </Radio>
              ))}
            </Flex>
          </Radio.Group>
        )}
      </Flex>
    </Drawer>
  );
};
