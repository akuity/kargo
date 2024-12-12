import { faGithub, faGitlab } from '@fortawesome/free-brands-svg-icons';
import { faChevronDown, faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Space } from 'antd';
import Dropdown from 'antd/es/dropdown/dropdown';
import { ReactNode, useMemo } from 'react';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { PromotionStep } from '@ui/gen/v1alpha1/generated_pb';

import { getPromotionState, getPromotionStepAlias, getPromotionStepConfig } from './plugin-helper';
import { PluginsInstallation } from './plugin-interfaces';

const getPullRequestLink = (
  promotionStep: PromotionStep,
  promotionStepOutput: Record<string, unknown>
) => {
  const stepConfig = getPromotionStepConfig(promotionStep);

  // plugins responsibility to keep up to date with actual promotion step config
  const provider = stepConfig?.provider as 'github' | 'gitlab';

  const repoURL = stepConfig?.repoURL as string;

  const prNumber = promotionStepOutput?.prNumber;

  if (typeof prNumber !== 'number' || !repoURL) {
    // TODO: Throw plugin identified error to handle it at plugin level
    throw new Error(
      `cannot generate pull request link, either promotion step didn't return proper output or promotion step config is corrupted`
    );
  }

  // url without slash at the end so it is easy to append path
  const url = repoURL?.endsWith?.('/') ? repoURL.slice(0, -1) : repoURL;

  if (provider === 'github') {
    return `${url}/pull/${prNumber}`;
  }

  if (provider === 'gitlab') {
    return `${url}/merge_requests/${prNumber}`;
  }

  throw new Error(`provider ${provider} not supported by this plugin`);
};

// TODO: refactor as it is getting bigger
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
    },
    Promotion: {
      shouldRender(opts) {
        return Boolean(opts.promotion?.spec?.steps?.find((step) => step?.uses === 'git-open-pr'));
      },
      render(props) {
        // array of [step-alias, deep-link]
        const deepLinks: string[][] = [];

        const promotionState = getPromotionState(props.promotion);

        for (const [idx, step] of Object.entries(props.promotion?.spec?.steps || [])) {
          if (step?.uses === 'git-open-pr') {
            const alias = getPromotionStepAlias(step, idx);

            try {
              const deepLink = getPullRequestLink(step, promotionState[alias]);

              deepLinks.push([alias, deepLink]);
            } catch {
              // TODO: failed to get deep link.. most probably due to invalid config/output
            }
          }
        }

        if (deepLinks?.length === 0) {
          return null;
        }

        if (deepLinks.length === 1) {
          // no need to have step context just show the deep link
          return (
            <a href={deepLinks[0][1]} target='_blank'>
              <FontAwesomeIcon icon={faCodePullRequest} />
            </a>
          );
        }

        return (
          <Dropdown
            menu={{
              items: deepLinks.map((deepLink, idx) => ({
                key: idx,
                label: (
                  <a key={idx} href={deepLink[1]} target='_blank'>
                    {deepLink[0]} - <FontAwesomeIcon icon={faCodePullRequest} />
                  </a>
                )
              }))
            }}
          >
            <a onClick={(e) => e.preventDefault()}>
              <Space>
                PRs
                <FontAwesomeIcon icon={faChevronDown} className='text-xs' />
              </Space>
            </a>
          </Dropdown>
        );
      }
    }
  }
};

export default plugin;
