import { useMemo } from 'react';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';

export const usePromotionEligibleFreight = (
  freight: Freight[],
  stage?: string,
  disabled?: boolean
) => {
  return useMemo(() => {
    if (disabled) {
      return {};
    }
    const availableFreight = !stage
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
