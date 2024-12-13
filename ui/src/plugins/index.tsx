import argocdPlugin from './argocd-plugin';
import { PluginsInstallation } from './plugin-interfaces';
import prPlugin from './pr-plugin';

const plugins: PluginsInstallation[] = [prPlugin, argocdPlugin];

export default plugins;
