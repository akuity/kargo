import { JSONSchema7 } from 'json-schema';

// IMPORTANT(Marvin9): this must be replaced with proper discovery mechanism
import argocdUpdateConfig from '@ui/gen/directives/argocd-update-config.json';
import argocdWaitConfig from '@ui/gen/directives/argocd-wait-config.json';
import composeOutputConfig from '@ui/gen/directives/compose-output-config.json';
import copyConfig from '@ui/gen/directives/copy-config.json';
import deleteConfig from '@ui/gen/directives/delete-config.json';
import failConfig from '@ui/gen/directives/fail-config.json';
import fileWriteConfig from '@ui/gen/directives/file-write-config.json';
import gitOverwriteConfig from '@ui/gen/directives/git-clear-config.json';
import gitCloneConfig from '@ui/gen/directives/git-clone-config.json';
import gitCommitConfig from '@ui/gen/directives/git-commit-config.json';
import gitMergePRConfig from '@ui/gen/directives/git-merge-pr-config.json';
import gitOpenPR from '@ui/gen/directives/git-open-pr-config.json';
import gitPushConfig from '@ui/gen/directives/git-push-config.json';
import gitTagConfig from '@ui/gen/directives/git-tag-config.json';
import gitWaitForPR from '@ui/gen/directives/git-wait-for-pr-config.json';
import githubPushConfig from '@ui/gen/directives/github-push-config.json';
import helmTemplateConfig from '@ui/gen/directives/helm-template-config.json';
import helmUpdateChartConfig from '@ui/gen/directives/helm-update-chart-config.json';
import httpConfig from '@ui/gen/directives/http-config.json';
import httpDownloadConfig from '@ui/gen/directives/http-download-config.json';
import jsonParseConfig from '@ui/gen/directives/json-parse-config.json';
import jsonUpdateConfig from '@ui/gen/directives/json-update-config.json';
import kustomizeBuildConfig from '@ui/gen/directives/kustomize-build-config.json';
import kustomizeSetImageConfig from '@ui/gen/directives/kustomize-set-image-config.json';
import ociDownloadConfig from '@ui/gen/directives/oci-download-config.json';
import ociPushConfig from '@ui/gen/directives/oci-push-config.json';
import setFreightAliasConfig from '@ui/gen/directives/set-freight-alias-config.json';
import setMetadataConfig from '@ui/gen/directives/set-metadata-config.json';
import tomlParseConfig from '@ui/gen/directives/toml-parse-config.json';
import tomlUpdateConfig from '@ui/gen/directives/toml-update-config.json';
import untarConfig from '@ui/gen/directives/untar-config.json';
import yamlMergeConfig from '@ui/gen/directives/yaml-merge-config.json';
import yamlParseConfig from '@ui/gen/directives/yaml-parse-config.json';
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
        config: argocdUpdateConfig as JSONSchema7
      },
      {
        identifier: 'argocd-wait',
        config: argocdWaitConfig as JSONSchema7
      },
      {
        identifier: 'copy',
        config: copyConfig as JSONSchema7
      },
      {
        identifier: 'compose-output',
        config: composeOutputConfig as JSONSchema7
      },
      {
        identifier: 'delete',
        unstable_icons: [],
        config: deleteConfig as JSONSchema7
      },
      {
        identifier: 'file-write',
        config: fileWriteConfig as JSONSchema7
      },
      {
        identifier: 'git-clone',
        config: gitCloneConfig as JSONSchema7
      },
      {
        identifier: 'git-push',
        config: gitPushConfig as JSONSchema7
      },
      {
        identifier: 'git-commit',
        config: gitCommitConfig as unknown as JSONSchema7
      },
      {
        identifier: 'git-open-pr',
        config: gitOpenPR as unknown as JSONSchema7
      },
      {
        identifier: 'git-merge-pr',
        config: gitMergePRConfig as JSONSchema7
      },
      {
        identifier: 'git-wait-for-pr',
        config: gitWaitForPR as unknown as JSONSchema7
      },
      {
        identifier: 'git-tag',
        config: gitTagConfig as JSONSchema7
      },
      {
        identifier: 'github-push',
        config: githubPushConfig as JSONSchema7
      },
      {
        identifier: 'yaml-merge',
        config: yamlMergeConfig as unknown as JSONSchema7
      },
      {
        identifier: 'yaml-parse',
        config: yamlParseConfig as JSONSchema7
      },
      {
        identifier: 'yaml-update',
        config: yamlUpdateConfig as unknown as JSONSchema7
      },
      {
        identifier: 'json-parse',
        config: jsonParseConfig as JSONSchema7
      },
      {
        identifier: 'json-update',
        config: jsonUpdateConfig as unknown as JSONSchema7
      },
      {
        identifier: 'toml-parse',
        config: tomlParseConfig as JSONSchema7
      },
      {
        identifier: 'toml-update',
        config: tomlUpdateConfig as unknown as JSONSchema7
      },
      {
        identifier: 'git-clear',
        config: gitOverwriteConfig as JSONSchema7
      },
      {
        identifier: 'helm-update-chart',
        config: helmUpdateChartConfig as JSONSchema7
      },
      {
        identifier: 'helm-template',
        config: helmTemplateConfig as JSONSchema7
      },
      {
        identifier: 'kustomize-build',
        config: kustomizeBuildConfig as JSONSchema7
      },
      {
        identifier: 'kustomize-set-image',
        config: kustomizeSetImageConfig as JSONSchema7
      },
      {
        identifier: 'http',
        config: httpConfig as JSONSchema7
      },
      {
        identifier: 'http-download',
        config: httpDownloadConfig as JSONSchema7
      },
      {
        identifier: 'oci-download',
        config: ociDownloadConfig as JSONSchema7
      },
      {
        identifier: 'oci-push',
        config: ociPushConfig as JSONSchema7
      },
      {
        identifier: 'untar',
        config: untarConfig as JSONSchema7
      },
      {
        identifier: 'set-freight-alias',
        config: setFreightAliasConfig as JSONSchema7
      },
      {
        identifier: 'set-metadata',
        config: setMetadataConfig as JSONSchema7
      },
      {
        identifier: 'fail',
        config: failConfig as JSONSchema7
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
