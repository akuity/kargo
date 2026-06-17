import { PluginsInstallation } from '../atoms/plugin-interfaces';

import promotionPlugin from './deep-link/promotion';
import promotionStepPlugin from './deep-link/promotion-step';

const plugin: PluginsInstallation = {
  DeepLinkPlugin: {
    PromotionStep: promotionStepPlugin,
    Promotion: promotionPlugin
  }
};

export default plugin;
