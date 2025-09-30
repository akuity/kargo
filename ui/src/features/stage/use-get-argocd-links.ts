import { useMemo } from 'react';

import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const useGetArgoCDLinks = (stage: Stage, argocdShard?: ArgoCDShard) =>
  useMemo(() => {
    const argocdContextKey = 'kargo.akuity.io/argocd-context';

    if (!argocdShard) {
      return [];
    }

    const argocdShardUrl = argocdShard?.url?.endsWith('/')
      ? argocdShard?.url?.slice(0, -1)
      : argocdShard?.url;

    const rawValues = stage.metadata?.annotations?.[argocdContextKey];

    if (!rawValues) {
      return [];
    }

    try {
      const parsedValues = JSON.parse(rawValues) as Array<{
        name: string;
        namespace: string;
      }>;

      return (
        parsedValues?.map(
          (parsedValue) =>
            `${argocdShardUrl}/applications/${parsedValue.namespace}/${parsedValue.name}`
        ) || []
      );
    } catch (e) {
      // deliberately do not crash
      // eslint-disable-next-line no-console
      console.error(e);

      return [];
    }
  }, [argocdShard, stage]);
