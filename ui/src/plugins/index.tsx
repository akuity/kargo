import argocdPlugin from './argocd-plugin/argocd-plugin';
import { PluginsInstallation } from './atoms/plugin-interfaces';
import prPlugin from './pr-plugin/pr-plugin';

const plugins: PluginsInstallation[] = [prPlugin, argocdPlugin];

export default plugins;
