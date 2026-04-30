import { faGithub, faGitlab } from '@fortawesome/free-brands-svg-icons';
import { faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { ReactNode, useMemo } from 'react';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { getPromotionStepConfig } from '@ui/plugins/atoms/plugin-helper';
import { DeepLinkPluginsInstallation } from '@ui/plugins/atoms/plugin-interfaces';

import { getGitProviderLink, getPullRequestLink, PR_STEP_TYPES } from '../get-pr-link';

const plugin: DeepLinkPluginsInstallation['PromotionStep'] = {
  shouldRender({ step, result }) {
    const isPRStep = PR_STEP_TYPES.includes(step?.uses || '');
    if (!isPRStep) {
      return false;
    }
    // git-open-pr: show link on success
    // git-wait-for-pr: show link on success OR running (since it outputs PR metadata while waiting)
    if (step?.uses === 'git-wait-for-pr') {
      return (
        result === PromotionDirectiveStepStatus.SUCCESS ||
        result === PromotionDirectiveStepStatus.RUNNING
      );
    }
    return result === PromotionDirectiveStepStatus.SUCCESS;
  },
  render(props) {
    const stepConfig = useMemo(() => getPromotionStepConfig(props.step), [props.step]);

    // plugins responsibility to keep up to date with actual promotion step config
    const provider = (stepConfig?.provider as 'github' | 'gitlab') || 'github';

    const repoURL = getGitProviderLink(props.output || {});

    const nodes: ReactNode[] = [];

    if (repoURL && provider === 'github') {
      nodes.push(
        <a href={repoURL} target='_blank'>
          <FontAwesomeIcon icon={faGithub} />
        </a>
      );
    } else if (repoURL && provider === 'gitlab') {
      nodes.push(
        <a href={repoURL} target='_blank'>
          <FontAwesomeIcon icon={faGitlab} />
        </a>
      );
    }

    const url = getPullRequestLink(props.output || {});

    if (url) {
      nodes.push(
        <a href={url} target='_blank'>
          <FontAwesomeIcon icon={faCodePullRequest} />
        </a>
      );
    }

    if (nodes.length === 0) return null;

    return <Flex gap={8}>{nodes}</Flex>;
  }
};

export default plugin;
