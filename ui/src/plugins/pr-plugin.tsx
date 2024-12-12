import { faGithub, faGitlab } from '@fortawesome/free-brands-svg-icons';
import { faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { ReactNode, useMemo } from 'react';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';

import { getPromotionStepConfig } from './plugin-helper';
import { PluginsInstallation } from './plugin-interfaces';

const plugin: PluginsInstallation = {
  DeepLinkPlugin: {
    PromotionStep: {
      shouldRender({ step, result }) {
        return result === PromotionDirectiveStepStatus.SUCCESS && step?.uses === 'git-open-pr';
      },
      render(props) {
        const stepConfig = useMemo(() => getPromotionStepConfig(props.step), [props.step]);

        // plugins responsibility to keep up to date with actual promotion step config
        const provider = stepConfig?.provider as 'github' | 'gitlab';

        const repoURL = stepConfig?.repoURL as string;

        const prNumber = props.output?.prNumber;

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

        if (typeof prNumber === 'number') {
          // url without slash at the end so it is easy to append path
          let url = repoURL.endsWith('/') ? repoURL.slice(0, -1) : repoURL;

          if (provider === 'github') {
            url = `${url}/pull/${prNumber}`;
          } else if (provider === 'gitlab') {
            url = `${url}/merge_requests/${prNumber}`;
          }

          nodes.push(
            <a href={url} target='_blank'>
              <FontAwesomeIcon icon={faCodePullRequest} />
            </a>
          );
        }

        return <Flex gap={8}>{nodes}</Flex>;
      }
    }
  }
};

export default plugin;
