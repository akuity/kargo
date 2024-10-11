import { Stage } from '@ui/gen/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

// context to bridge directives and pipelines UI

export const willStagePromotionOpenPR = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps || []).some((g) =>
    // TODO(Marvin9): find better way than to hardcode
    g?.uses?.includes('git-open-pr')
  );

export const getPromotionArgoCDApps = (stage: Stage): string[] => {
  let apps: string[] = [];
  const config = stage?.spec?.promotionTemplate?.spec?.steps?.find((step) =>
    step?.uses?.includes('argocd-update')
  )?.config?.raw; /* now type is detached with dynamic config */

  if (config) {
    try {
      const data = JSON.parse(
        decodeRawData({ result: { case: 'raw', value: config } })
      ) as /* from docs */ {
        apps?: Array<{
          name?: string;
        }>;
      };
      apps = data?.apps?.map((app) => app?.name || '').filter(Boolean) || [];
    } catch {
      // explicitly ignore, no need to crash or anything, this is UI addon
    }
  }

  return apps;
};
