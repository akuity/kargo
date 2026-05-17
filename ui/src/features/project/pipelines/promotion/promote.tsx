import { useMutation as useConnectMutation } from '@connectrpc/connect-query';
import { faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Button, Drawer, Flex, Input } from 'antd';
import classNames from 'classnames';
import { useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getCurrentFreight } from '@ui/features/common/utils';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { promoteDownstream } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { useGetStageAutoPromotionCandidates, usePromoteToStage } from '@ui/gen/api/v2/core/core';

import { useDictionaryContext } from '../context/dictionary-context';
import { isStageControlFlow } from '../nodes/stage-meta-utils';

import {
  autoPromotionHoldStateActive,
  getAutoPromotionCandidateName,
  getAutoPromotionHold,
  originLabel
} from './auto-promotion';
import { FreightDetails } from './freight-details';
import styles from './promote.module.less';

type PromoteProps = ModalComponentProps & {
  stage: Stage;
  freight: Freight;
};

export const Promote = (props: PromoteProps) => {
  const actionContext = useActionContext();
  const navigate = useNavigate();
  const { promoteTabs } = useExtensionsContext();
  const [reason, setReason] = useState('');

  const dictionaryContext = useDictionaryContext();

  const isDownstreamPromotion =
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM || isStageControlFlow(props.stage);

  const freightAlias = props.freight?.alias;
  const stageName = props.stage?.metadata?.name || '';
  const projectName = props.stage?.metadata?.namespace || '';
  const freightName = props.freight?.metadata?.name || '';

  const currentFreightOnStage = useMemo(() => getCurrentFreight(props.stage)[0], [props.stage]);

  const shouldCheckAutoPromotionCandidate = Boolean(
    projectName && stageName && !isDownstreamPromotion && props.stage?.status?.autoPromotionEnabled
  );
  const autoPromotionCandidatesQuery = useGetStageAutoPromotionCandidates(
    projectName || '',
    stageName || '',
    {
      query: {
        enabled: shouldCheckAutoPromotionCandidate
      }
    }
  );

  const isCheckingAutoPromotionCandidate =
    shouldCheckAutoPromotionCandidate && autoPromotionCandidatesQuery.isLoading;
  const candidateName = getAutoPromotionCandidateName(
    autoPromotionCandidatesQuery.data?.data?.candidates,
    props.freight
  );
  const selectedOriginLabel = originLabel(props.freight?.origin);
  const isPromotingOlderThanCandidate = Boolean(candidateName && candidateName !== freightName);
  const activeHold = getAutoPromotionHold(props.stage, props.freight?.origin);
  const willResumeOnSuccess = Boolean(
    activeHold?.state === autoPromotionHoldStateActive && candidateName === freightName
  );

  const promoteActionMutation = usePromoteToStage({
    mutation: {
      onSuccess: (response) => {
        // navigate
        navigate(
          generatePath(paths.promotion, {
            name: projectName,
            promotionId: response.data?.metadata?.name
          })
        );

        actionContext?.cancel();
      }
    }
  });

  const promoteDownstreamActionMutation = useConnectMutation(promoteDownstream, {
    onSuccess: () => {
      // navigate
      navigate(
        generatePath(paths.project, {
          name: projectName
        })
      );

      actionContext?.cancel();
    }
  });

  const onPromote = () => {
    const downstreamPayload = {
      stage: stageName,
      project: projectName,
      freight: freightName
    };

    if (isDownstreamPromotion) {
      promoteDownstreamActionMutation.mutate(downstreamPayload);
      return;
    }

    promoteActionMutation.mutate({
      stage: stageName,
      project: projectName,
      data: {
        freight: freightName,
        expectedAutoCandidate: candidateName || undefined,
        reason: isPromotingOlderThanCandidate ? reason.trim() || undefined : undefined
      }
    });
  };

  let promotingTo = stageName || '';

  if (isDownstreamPromotion) {
    promotingTo = [...(dictionaryContext?.subscribersByStage?.[promotingTo] || [])].join(', ');
  }

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      title={
        <Flex align='center'>
          Promote {freightAlias} to {promotingTo}
        </Flex>
      }
      size='large'
      width={'1400px'}
      footer={
        <Button
          size='large'
          className={classNames(styles['promote-btn'], 'ml-auto mt-5')}
          icon={<FontAwesomeIcon icon={faTruckArrowRight} />}
          onClick={onPromote}
          loading={
            isCheckingAutoPromotionCandidate ||
            promoteActionMutation.isPending ||
            promoteDownstreamActionMutation.isPending
          }
          disabled={isCheckingAutoPromotionCandidate}
        >
          {isCheckingAutoPromotionCandidate
            ? 'Checking auto-promotion'
            : isDownstreamPromotion
              ? 'Promote to downstream'
              : isPromotingOlderThanCandidate
                ? 'Roll back and pause auto-promotion'
                : 'Promote'}
        </Button>
      }
    >
      <div className='-mt-4'>
        {isCheckingAutoPromotionCandidate && (
          <Alert
            className='mb-4'
            showIcon
            type='info'
            message='Checking the current auto-promotion candidate.'
            description='Promotion is disabled until Kargo can show whether this will pause auto-promotion.'
          />
        )}

        {isPromotingOlderThanCandidate && (
          <div className='mb-4'>
            <Alert
              showIcon
              type='warning'
              message={`This is older than the current auto-promotion candidate ${candidateName}.`}
              description={`Auto-promotion for ${selectedOriginLabel} will pause if this Promotion succeeds.`}
            />
            <Input.TextArea
              className='mt-3'
              placeholder='Reason (optional)'
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              maxLength={1024}
              showCount
              autoSize={{ minRows: 2, maxRows: 4 }}
            />
          </div>
        )}

        {willResumeOnSuccess && (
          <Alert
            className='mb-4'
            showIcon
            type='info'
            message={`This is the current auto-promotion candidate for ${selectedOriginLabel}.`}
            description='Auto-promotion will resume for this origin if the Promotion succeeds.'
          />
        )}

        <FreightDetails
          freight={props.freight}
          comparison={{ currentFreight: currentFreightOnStage }}
          additionalTabs={promoteTabs.map((data, index) => ({
            children: <data.component freight={props.freight} stage={props.stage} />,
            key: String(data.label + index),
            label: data.label,
            icon: data.icon
          }))}
        />
      </div>
    </Drawer>
  );
};
