import { PluginsInstallation } from '../atoms/plugin-interfaces';

import promotionDeepLinkPlugin from './deep-link/promotion';
import promotionStepDeepLinkPlugin from './deep-link/promotion-step';

const plugin: PluginsInstallation = {
  DeepLinkPlugin: {
    PromotionStep: promotionStepDeepLinkPlugin,
    Promotion: promotionDeepLinkPlugin
  }
};

export default plugin;
