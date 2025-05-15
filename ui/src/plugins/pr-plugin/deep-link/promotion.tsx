import { faChevronDown, faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Space } from 'antd';

import { getPromotionState, getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';
import { DeepLinkPluginsInstallation } from '@ui/plugins/atoms/plugin-interfaces';

import { getPullRequestLink } from '../get-pr-link';

const plugin: DeepLinkPluginsInstallation['Promotion'] = {
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
          const deepLink = getPullRequestLink(promotionState[alias]);

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
};

export default plugin;
