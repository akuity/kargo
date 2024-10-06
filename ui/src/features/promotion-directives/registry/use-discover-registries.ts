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
  faFileCode,
  faFileEdit,
  faFileImage,
  faHammer,
  faHeart,
  faRedoAlt,
  faSyncAlt,
  faUpDown,
  faWrench
} from '@fortawesome/free-solid-svg-icons';

import { PromotionDirectivesRegistry } from './types';

export const useDiscoverPromotionDirectivesRegistries = (): PromotionDirectivesRegistry => {
  // at this point, we have only built-in runners and available out of the box
  // for that reason, we don't need to delegate discovery logic, this is the source of truth
  // when we actually starts accepting external promotion directives registry, this must be the only place to care about
  return {
    runners: [
      {
        identifier: 'argocd-update',
        unstable_icons: [faUpDown, faHeart]
      },
      {
        identifier: 'copy',
        unstable_icons: [faCopy]
      },
      {
        identifier: 'git-clone',
        unstable_icons: [faCodeBranch, faClone]
      },
      {
        identifier: 'git-commit',
        unstable_icons: [faCheck, faCodeBranch]
      },
      {
        identifier: 'git-open-pr',
        unstable_icons: [faCodePullRequest, faFileCode]
      },
      {
        identifier: 'git-wait-for-pr',
        unstable_icons: [faClock, faCodePullRequest]
      },
      {
        identifier: 'git-push',
        unstable_icons: [faArrowUp, faCloudUploadAlt]
      },
      {
        identifier: 'git-clear',
        unstable_icons: [faRedoAlt, faCodeBranch]
      },
      {
        identifier: 'helm-update-chart',
        unstable_icons: [faSyncAlt, faChartLine]
      },
      {
        identifier: 'helm-update-image',
        unstable_icons: [faSyncAlt, faFileImage]
      },
      {
        identifier: 'helm-template',
        unstable_icons: [faFileCode, faDraftingCompass]
      },
      {
        identifier: 'kustomize-build',
        unstable_icons: [faWrench, faHammer]
      },
      {
        identifier: 'kustomize-set-image',
        unstable_icons: [faFileImage, faFileEdit]
      }
    ]
  };
};
