import { faBoltLightning, faCircleNotch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Typography } from 'antd';
import { ItemType } from 'antd/es/menu/interface';
import { useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useGetUpstreamFreight } from '@ui/features/project/pipelines/nodes/use-get-upstream-freight';
import { stageHasAutoPromotionHold } from '@ui/features/project/pipelines/promotion/auto-promotion';
import { ResumeAutoPromotionDrawer } from '@ui/features/project/pipelines/promotion/resume-auto-promotion-drawer';
import { useManualApprovalModal } from '@ui/features/project/pipelines/promotion/use-manual-approval-modal';
import { promoteToStage, queryFreightsRest } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

export const useGetPromotionDropdownItems = (stage: Stage) => {
  const [resumeAutoPromotionOpen, setResumeAutoPromotionOpen] = useState(false);

  const projectName = stage?.metadata?.namespace || '';
  const stageName = stage?.metadata?.name || '';

  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  const navigate = useNavigate();

  const totalSubscribersToThisStage = dictionaryContext?.subscribersByStage?.[stageName]?.size || 0;

  const controlFlow = isStageControlFlow(stage);
  const hasAutoPromotionHold = stageHasAutoPromotionHold(stage);

  const upstreamFreights = useGetUpstreamFreight(stage);

  const queryFreightMutation = useMutation({
    mutationFn: (payload: { project: string; stage: string }) =>
      queryFreightsRest(payload.project, { stage: payload.stage })
  });

  const showManualApproveModal = useManualApprovalModal();

  const ensureEligibilityBeforeAction = async ({
    freight,
    onSuccess
  }: {
    freight: string | undefined;
    onSuccess: (eligibleFreight: string) => void;
  }) => {
    if (!freight) {
      return;
    }

    const freightResponse = await queryFreightMutation.mutateAsync({
      project: projectName,
      stage: stageName
    });

    const isEligible = Boolean(
      freightResponse?.data.groups?.['']?.items?.find((item) => item?.metadata?.name === freight)
    );

    if (isEligible) {
      onSuccess(freight);
    } else {
      showManualApproveModal({
        freight,
        projectName,
        stage: stageName,
        onApprove: () => onSuccess(freight)
      });
    }
  };

  const handlePromoteFromUpstream = (freight?: string) => {
    ensureEligibilityBeforeAction({
      freight,
      onSuccess: (eligibleFreight) =>
        navigate(
          generatePath(paths.promote, {
            name: projectName,
            freight: eligibleFreight,
            stage: stageName
          })
        )
    });
  };

  const promoteMutation = useMutation({
    mutationFn: (payload: { project: string; stage: string; freight: string }) =>
      promoteToStage(payload.project, payload.stage, { freight: payload.freight }),
    onSuccess: (response) => {
      navigate(
        generatePath(paths.promotion, {
          name: projectName,
          promotionId: response.data?.metadata?.name
        })
      );
    }
  });

  const handleInstantPromoteFromUpstream = (freight?: string) => {
    ensureEligibilityBeforeAction({
      freight,
      onSuccess: (eligibleFreight) =>
        promoteMutation.mutate({
          stage: stageName,
          project: projectName,
          freight: eligibleFreight
        })
    });
  };

  const dropdownItems: ItemType[] = [];

  if (!controlFlow) {
    dropdownItems.push({
      key: 'promote',
      label: 'Promote',
      onClick: () => actionContext?.actPromote(IAction.PROMOTE, stage)
    });

    if (hasAutoPromotionHold) {
      dropdownItems.push({
        key: 'resume-auto-promotion',
        label: 'Resume auto-promotion',
        onClick: () => setResumeAutoPromotionOpen(true)
      });
    }
  }

  if (controlFlow || totalSubscribersToThisStage > 1) {
    dropdownItems.push({
      key: 'promote-downstream',
      label: 'Promote to downstream',
      onClick: () => actionContext?.actPromote(IAction.PROMOTE_DOWNSTREAM, stage)
    });
  }
  const hasUpstreamFreights = (upstreamFreights?.length || 0) > 0;
  const hasMultipleUpstreamFreights = (upstreamFreights?.length || 0) > 1;

  if (hasUpstreamFreights && !controlFlow) {
    dropdownItems.push({
      key: 'upstream-freight-promo',
      label: 'Promote from upstream',
      onClick: () => {
        if (hasMultipleUpstreamFreights) {
          return;
        }

        const freight = upstreamFreights?.[0]?.name;

        handlePromoteFromUpstream(freight);
      },
      children: hasMultipleUpstreamFreights
        ? upstreamFreights?.map((upstreamFreight) => ({
            key: upstreamFreight?.name || '',
            label: upstreamFreight?.origin?.name,
            onClick: () => handlePromoteFromUpstream(upstreamFreight?.name)
          }))
        : undefined
    });

    dropdownItems.push({
      key: 'quick-promote-upstream-freight-promo',
      label: (
        <>
          {promoteMutation.isPending ? (
            <FontAwesomeIcon icon={faCircleNotch} className='mr-1' spin />
          ) : (
            <Typography.Text type='danger' className='mr-2'>
              <FontAwesomeIcon icon={faBoltLightning} />
            </Typography.Text>
          )}
          Instant promote from upstream
        </>
      ),
      onClick: () => {
        if (hasMultipleUpstreamFreights) {
          return;
        }

        const freight = upstreamFreights?.[0]?.name;

        handleInstantPromoteFromUpstream(freight);
      },
      children: hasMultipleUpstreamFreights
        ? upstreamFreights?.map((upstreamFreight) => ({
            key: upstreamFreight?.name || '',
            label: upstreamFreight?.origin?.name,
            onClick: () => handleInstantPromoteFromUpstream(upstreamFreight?.name)
          }))
        : undefined
    });
  }

  return {
    dropdownItems,
    resumeAutoPromotionDrawer: (
      <ResumeAutoPromotionDrawer
        stage={stage}
        open={resumeAutoPromotionOpen}
        onClose={() => setResumeAutoPromotionOpen(false)}
      />
    )
  };
};
