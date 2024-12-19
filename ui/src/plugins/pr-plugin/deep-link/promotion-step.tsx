import { faGithub, faGitlab } from '@fortawesome/free-brands-svg-icons';
import { faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { ReactNode, useMemo } from 'react';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { getPromotionStepConfig } from '@ui/plugins/atoms/plugin-helper';
import { DeepLinkPluginsInstallation } from '@ui/plugins/atoms/plugin-interfaces';

import { getPullRequestLink } from '../get-pr-link';

const plugin: DeepLinkPluginsInstallation['PromotionStep'] = {
  shouldRender({ step, result }) {
    return result === PromotionDirectiveStepStatus.SUCCESS && step?.uses === 'git-open-pr';
  },
  render(props) {
    const stepConfig = useMemo(() => getPromotionStepConfig(props.step), [props.step]);

    // plugins responsibility to keep up to date with actual promotion step config
    const provider = (stepConfig?.provider as 'github' | 'gitlab') || 'github';

    const repoURL = stepConfig?.repoURL as string;

    const nodes: ReactNode[] = [];

    if (provider === 'github') {
      nodes.push(
        <a href={repoURL} target='_blank'>
          <FontAwesomeIcon icon={faGithub} />
        </a>
      );
    } else if (provider === 'gitlab') {
      nodes.push(
        <a href={repoURL} target='_blank'>
          <FontAwesomeIcon icon={faGitlab} />
        </a>
      );
    }

    const url = getPullRequestLink(props.step, props.output || {});

    nodes.push(
      <a href={url} target='_blank'>
        <FontAwesomeIcon icon={faCodePullRequest} />
      </a>
    );

    return <Flex gap={8}>{nodes}</Flex>;
  }
};

export default plugin;
