import { IconDefinition } from '@fortawesome/fontawesome-svg-core';
import { faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import {
  isPromotionStepStatusTerminal,
  PromotionStepStatusPhase
} from '@ui/features/common/promotion-status/utils';
import { getPromotionOutputsByStepAlias } from '@ui/features/stage/utils/promotion';
import { Stage } from '@ui/gen/api/v2/models';
import { getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';
import { getPullRequestLink } from '@ui/plugins/pr-plugin/get-pr-link';

import { useCurrentPromotion } from './stage-meta-utils';

// StageWaiting is one thing holding a promotion up, as the Status column has
// to state it: a few words, a mark, and somewhere to go and deal with it.
export type StageWaiting = {
  // Short enough to sit on one line in half a card, which is all the column
  // is. The full wording, where there is more of it, goes in the title.
  label: string;
  title: string;
  icon: IconDefinition;
  // to is the Promotion holding things up. The label leads there, exactly as
  // the phase it stands in for does while a Stage promotes.
  to?: string;
  // href is somewhere outside Kargo -- a pull request -- and earns a line of
  // its own under the label. Nothing waits on anything external without one.
  href?: string;
  linkLabel?: string;
};

// The step kind that actually waits on a pull request. git-open-pr succeeds as
// soon as the PR is opened and never represents a waiting state.
const prStepKind = 'git-wait-for-pr';

// useStageWaiting reports what the Stage's current promotion is waiting on, if
// anything. A promotion pauses on one step at a time, so there is at most one
// answer: the first waiting step in the order the steps run.
export const useStageWaiting = (stage: Stage): StageWaiting | undefined => {
  const { promotion } = useCurrentPromotion(stage);
  const { promotionStepExtensions } = useExtensionsContext();

  const steps = promotion?.spec?.steps;

  if (!promotion || !steps) {
    return undefined;
  }

  const stepStatus = (index: number) => promotion?.status?.stepExecutionMetadata?.[index]?.status;

  const promotionPath = generatePath(paths.promotion, {
    name: stage?.metadata?.namespace || '',
    promotionId: promotion?.metadata?.name || ''
  });

  for (const [index, step] of steps.entries()) {
    const uses = (step as { uses?: string })?.uses;

    if (uses === prStepKind) {
      // The link comes from the step's own output, so it exists only once the
      // pull request has been opened. The status is checked as well: a merged
      // or failed PR step has reached a terminal status and is no longer
      // holding anything up.
      const outputs = (getPromotionOutputsByStepAlias(promotion)?.[
        getPromotionStepAlias(step, index)
      ] || {}) as Record<string, unknown>;

      const url = getPullRequestLink(outputs);

      if (url && !isPromotionStepStatusTerminal(stepStatus(index))) {
        return {
          label: 'Waiting for PR',
          title: 'Waiting for pull request approval',
          icon: faCodePullRequest,
          to: promotionPath,
          href: url,
          linkLabel: 'Open PR'
        };
      }

      continue;
    }

    // Anything else that waits does so because an extension says it does.
    const extension = promotionStepExtensions.find(
      (ext) => ext.waitingLabel && ext.identifier === uses
    );

    if (extension && stepStatus(index) === PromotionStepStatusPhase.RUNNING) {
      // No second line: the label already leads to the Promotion, which is
      // the only place an extension's waiting step can be dealt with.
      return {
        label: extension.waitingLabel!.label,
        title: extension.waitingLabel!.label,
        icon: extension.waitingLabel!.icon,
        to: promotionPath
      };
    }
  }

  return undefined;
};
