import { faChevronDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Space, Tooltip } from 'antd';

import { getPromotionState, getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';
import { DeepLinkPluginsInstallation } from '@ui/plugins/atoms/plugin-interfaces';

import { getOutputLinks, StepOutputLink } from '../output-links';
import { resolveIcon } from '../resolve-icon';

function collectLinks(
  props: Parameters<NonNullable<DeepLinkPluginsInstallation['Promotion']>['render']>[0]
): { alias: string; link: StepOutputLink }[] {
  try {
    const state = getPromotionState(props.promotion);
    const result: { alias: string; link: StepOutputLink }[] = [];

    for (const [idx, step] of Object.entries(props.promotion?.spec?.steps || [])) {
      const alias = getPromotionStepAlias(step, idx);
      try {
        const links = getOutputLinks(state[alias] as Record<string, unknown>);
        for (const link of links) {
          result.push({ alias, link });
        }
      } catch {
        // malformed step output - skip
      }
    }

    return result;
  } catch {
    return [];
  }
}

const plugin: DeepLinkPluginsInstallation['Promotion'] = {
  shouldRender(opts) {
    try {
      const state = getPromotionState(opts.promotion);
      return Object.values(state || {}).some(
        (stepOutput) => getOutputLinks(stepOutput as Record<string, unknown>).length > 0
      );
    } catch {
      return false;
    }
  },
  render(props) {
    const allLinks = collectLinks(props);

    if (allLinks.length === 0) return null;

    if (allLinks.length === 1) {
      const { link } = allLinks[0];
      return (
        <Tooltip title={link.tooltip || link.label}>
          <a href={link.url} target='_blank' rel='noreferrer'>
            <FontAwesomeIcon icon={resolveIcon(link.icon)} />
          </a>
        </Tooltip>
      );
    }

    return (
      <Dropdown
        menu={{
          items: allLinks.map(({ alias, link }, idx) => ({
            key: idx,
            label: (
              <a href={link.url} target='_blank' rel='noreferrer'>
                {link.label || alias} <FontAwesomeIcon icon={resolveIcon(link.icon)} />
              </a>
            )
          }))
        }}
      >
        <a onClick={(e) => e.preventDefault()}>
          <Space>
            Links
            <FontAwesomeIcon icon={faChevronDown} className='text-xs' />
          </Space>
        </a>
      </Dropdown>
    );
  }
};

export default plugin;
