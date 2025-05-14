import { useQuery } from '@connectrpc/connect-query';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const usePromotionEligibleFreight = (project: string) => {
  const actionContext = useActionContext();

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  const getPromotionEligibleFreightQuery = useQuery(
    queryFreight,
    {
      project,
      stage: actionContext?.action?.stage?.metadata?.name
    },
    {
      enabled: isPromotionMode
    }
  );

  let promotionEligibleFreight =
    getPromotionEligibleFreightQuery?.data?.groups?.['']?.freight || [];

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
