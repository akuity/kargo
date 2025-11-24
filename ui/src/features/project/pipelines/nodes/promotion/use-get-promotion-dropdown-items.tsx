import { useMutation } from '@connectrpc/connect-query';
import { faBoltLightning, faCircleNotch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';
import { ItemType } from 'antd/es/menu/interface';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useGetUpstreamFreight } from '@ui/features/project/pipelines/nodes/use-get-upstream-freight';
import { useManualApprovalModal } from '@ui/features/project/pipelines/promotion/use-manual-approval-modal';
import {
  promoteToStage,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const useGetPromotionDropdownItems = (stage: Stage) => {
  const projectName = stage?.metadata?.namespace || '';
  const stageName = stage?.metadata?.name || '';

  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  const navigate = useNavigate();

  const totalSubscribersToThisStage = dictionaryContext?.subscribersByStage?.[stageName]?.size || 0;

  const controlFlow = isStageControlFlow(stage);

  const upstreamFreights = useGetUpstreamFreight(stage);

  const queryFreightMutation = useMutation(queryFreight);

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
      freightResponse?.groups?.['']?.freight?.find((item) => item?.metadata?.name === freight)
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

  const promoteMutation = useMutation(promoteToStage, {
    onSuccess: (response) => {
      navigate(
        generatePath(paths.promotion, {
          name: projectName,
          promotionId: response.promotion?.metadata?.name
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

  if (hasUpstreamFreights) {
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
            key: upstreamFreight?.name,
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
            key: upstreamFreight?.name,
            label: upstreamFreight?.origin?.name,
            onClick: () => handleInstantPromoteFromUpstream(upstreamFreight?.name)
          }))
        : undefined
    });
  }

  return dropdownItems;
};
