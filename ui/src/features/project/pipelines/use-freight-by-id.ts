import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

export const useFreightById = (freights: Freight[]): Record<string, Freight> =>
  useMemo(() => {
    const freightById: Record<string, Freight> = {};

    for (const freight of freights) {
      freightById[freight?.metadata?.name || ''] = freight;
    }

    return freightById;
  }, [freights]);
