import { PluginsInstallation } from './atoms/plugin-interfaces';
import outputLinksPlugin from './output-links-plugin/output-links-plugin';
import prPlugin from './pr-plugin/pr-plugin';

const plugins: PluginsInstallation[] = [prPlugin, outputLinksPlugin];

export default plugins;
