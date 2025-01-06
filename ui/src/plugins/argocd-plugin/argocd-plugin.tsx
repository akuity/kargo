import { PluginsInstallation } from '../atoms/plugin-interfaces';

import promotionDeepLinkPlugin from './deep-link/promotion';

const plugin: PluginsInstallation = {
  DeepLinkPlugin: {
    Promotion: promotionDeepLinkPlugin
  }
};

export default plugin;
