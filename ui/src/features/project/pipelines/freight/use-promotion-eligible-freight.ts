import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';

export const usePromotionEligibleFreight = (project: string) => {
  const actionContext = useActionContext();

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  const getPromotionEligibleFreightQuery = useQueryFreightsRest(
    project,
    {
      stage: actionContext?.action?.stage?.metadata?.name
    },
    { query: { enabled: isPromotionMode } }
  );

  let promotionEligibleFreight =
    getPromotionEligibleFreightQuery?.data?.data?.groups?.['']?.items || [];

  if (actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM) {
    promotionEligibleFreight = promotionEligibleFreight.filter(
      (f) => f?.status?.verifiedIn?.[actionContext?.action?.stage?.metadata?.name || '']
    );
  }

  return {
    ...getPromotionEligibleFreightQuery,
    promotionEligibleFreight
  };
};
