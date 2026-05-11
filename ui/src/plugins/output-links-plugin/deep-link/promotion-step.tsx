import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tooltip } from 'antd';

import { DeepLinkPluginsInstallation } from '@ui/plugins/atoms/plugin-interfaces';

import { getOutputLinks } from '../output-links';
import { iconExists, resolveIcon } from '../resolve-icon';

const plugin: DeepLinkPluginsInstallation['PromotionStep'] = {
  shouldRender({ output }) {
    return getOutputLinks(output || {}).length > 0;
  },
  render(props) {
    const links = getOutputLinks(props.output || {});
    if (links.length === 0) return null;

    return (
      <Flex gap={8}>
        {links.map((link, idx) => (
          <Tooltip key={idx} title={link.tooltip || link.label}>
            <a href={link.url} target='_blank' rel='noreferrer'>
              <FontAwesomeIcon icon={resolveIcon(link.icon)} />
              {!iconExists(link.icon) && link.label && <span className='ml-1'>{link.label}</span>}
            </a>
          </Tooltip>
        ))}
      </Flex>
    );
  }
};

export default plugin;
