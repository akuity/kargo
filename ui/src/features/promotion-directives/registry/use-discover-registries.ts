import {
  faArrowUp,
  faChartLine,
  faCheck,
  faClock,
  faClone,
  faCloudUploadAlt,
  faCodeBranch,
  faCodePullRequest,
  faCopy,
  faDraftingCompass,
  faEdit,
  faExchangeAlt,
  faFileCode,
  faFileEdit,
  faFileImage,
  faGlobe,
  faHammer,
  faHeart,
  faNetworkWired,
  faRedoAlt,
  faSyncAlt,
  faUpDown,
  faUpload,
  faWrench
} from '@fortawesome/free-solid-svg-icons';
import { JSONSchema7 } from 'json-schema';

// IMPORTANT(Marvin9): this must be replaced with proper discovery mechanism
import argocdUpdateConfig from '@ui/gen/directives/argocd-update-config.json';
import copyConfig from '@ui/gen/directives/copy-config.json';
import gitOverwriteConfig from '@ui/gen/directives/git-clear-config.json';
import gitCloneConfig from '@ui/gen/directives/git-clone-config.json';
import gitCommitConfig from '@ui/gen/directives/git-commit-config.json';
import gitOpenPR from '@ui/gen/directives/git-open-pr-config.json';
import gitPushConfig from '@ui/gen/directives/git-push-config.json';
import gitWaitForPR from '@ui/gen/directives/git-wait-for-pr-config.json';
import helmTemplateConfig from '@ui/gen/directives/helm-template-config.json';
import helmUpdateChartConfig from '@ui/gen/directives/helm-update-chart-config.json';
import helmUpdateImageConfig from '@ui/gen/directives/helm-update-image-config.json';
import httpConfig from '@ui/gen/directives/http-config.json';
import kustomizeBuildConfig from '@ui/gen/directives/kustomize-build-config.json';
import kustomizeSetImageConfig from '@ui/gen/directives/kustomize-set-image-config.json';
import yamlUpdateConfig from '@ui/gen/directives/yaml-update-config.json';

import { PromotionDirectivesRegistry } from './types';

export const useDiscoverPromotionDirectivesRegistries = (): PromotionDirectivesRegistry => {
  // at this point, we have only built-in runners and available out of the box
  // for that reason, we don't need to delegate discovery logic, this is the source of truth
  // when we actually starts accepting external promotion directives registry, this must be the only place to care about
  const registry: PromotionDirectivesRegistry = {
    runners: [
      {
        identifier: 'argocd-update',
        unstable_icons: [faUpDown, faHeart],
        config: argocdUpdateConfig as JSONSchema7
      },
      {
        identifier: 'copy',
        unstable_icons: [faCopy],
        config: copyConfig as JSONSchema7
      },
      {
        identifier: 'git-clone',
        unstable_icons: [faCodeBranch, faClone],
        config: gitCloneConfig as JSONSchema7
      },
      {
        identifier: 'git-push',
        unstable_icons: [faArrowUp, faCloudUploadAlt, faUpload],
        config: gitPushConfig as JSONSchema7
      },
      {
        identifier: 'git-commit',
        unstable_icons: [faCheck, faCodeBranch],
        config: gitCommitConfig as unknown as JSONSchema7
      },
      {
        identifier: 'git-open-pr',
        unstable_icons: [faCodePullRequest, faFileCode],
        config: gitOpenPR as unknown as JSONSchema7
      },
      {
        identifier: 'git-wait-for-pr',
        unstable_icons: [faClock, faCodePullRequest],
        config: gitWaitForPR as unknown as JSONSchema7
      },
      {
        identifier: 'yaml-update',
        config: yamlUpdateConfig as unknown as JSONSchema7,
        unstable_icons: [faFileCode, faSyncAlt, faEdit]
      },
      {
        identifier: 'git-push',
        unstable_icons: [faArrowUp, faCloudUploadAlt],
        config: gitPushConfig as unknown as JSONSchema7
      },
      {
        identifier: 'git-clear',
        unstable_icons: [faRedoAlt, faCodeBranch],
        config: gitOverwriteConfig as JSONSchema7
      },
      {
        identifier: 'helm-update-chart',
        unstable_icons: [faSyncAlt, faChartLine],
        config: helmUpdateChartConfig as JSONSchema7
      },
      {
        identifier: 'helm-update-image',
        unstable_icons: [faSyncAlt, faFileImage],
        config: helmUpdateImageConfig as JSONSchema7
      },
      {
        identifier: 'helm-template',
        unstable_icons: [faFileCode, faDraftingCompass],
        config: helmTemplateConfig as JSONSchema7
      },
      {
        identifier: 'kustomize-build',
        unstable_icons: [faWrench, faHammer],
        config: kustomizeBuildConfig as JSONSchema7
      },
      {
        identifier: 'kustomize-set-image',
        unstable_icons: [faFileImage, faFileEdit],
        config: kustomizeSetImageConfig as JSONSchema7
      },
      {
        identifier: 'http',
        unstable_icons: [faGlobe, faNetworkWired, faExchangeAlt],
        config: httpConfig as JSONSchema7
      }
    ]
  };

  registry.runners = registry.runners.map((runner) => {
    delete runner.config.$ref;
    delete runner.config.$schema;
    delete runner.config.title;
    return runner;
  });

  return registry;
};
