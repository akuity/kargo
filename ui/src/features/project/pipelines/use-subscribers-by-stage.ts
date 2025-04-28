import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const useSubscribersByStage = (stages: Stage[]): Record<string, Set<string>> => {
  const subscribersByStage: Record<string, Set<string>> = {};

  for (const stage of stages) {
    const childStageName = stage?.metadata?.name || '';
    for (const freight of stage?.spec?.requestedFreight || []) {
      if (!freight?.sources?.direct) {
        for (const source of freight?.sources?.stages || []) {
          if (!subscribersByStage[source]) {
            subscribersByStage[source] = new Set<string>();
          }

          subscribersByStage[source].add(childStageName);
        }
      }
    }
  }

  return subscribersByStage;
};
