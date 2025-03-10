import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightTimelineAction } from '../types';

export const usePromotionEligibleFreight = (
  freight: Freight[],
  action?: FreightTimelineAction,
  stage?: string,
  disabled?: boolean
) => {
  return useMemo(() => {
    if (disabled || !action) {
      return {};
    }
    const availableFreight =
      action === FreightTimelineAction.Promote || !stage
        ? freight
        : // if promoting subscribers, only include freight that has been verified in the promoting stage
          freight.filter((f) => !!f?.status?.verifiedIn[stage]);

    const pe: { [key: string]: boolean } = {};
    ((availableFreight as Freight[]) || []).forEach((f: Freight) => {
      pe[f?.metadata?.name || ''] = true;
    });
    return pe;
  }, [freight]);
};
