import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';

import { SegmentLabel } from '@ui/features/common/segment-label';
import { Secret } from '@ui/gen/v1alpha1/types_pb';

import { CredentialTypeLabelKey, CredentialsDataKey, CredentialsType } from './types';

export const typeLabel = (type: CredentialsType) => (
  <SegmentLabel icon={iconForCredentialsType(type)}>{type.toUpperCase()}</SegmentLabel>
);

export const iconForCredentialsType = (type: CredentialsType) => {
  switch (type) {
    case 'git':
      return faGit;
    case 'helm':
      return faAnchor;
    case 'image':
      return faDocker;
    default:
      return faQuestionCircle;
  }
};

export const labelForKey = (s: string) =>
  s
    .split('')
    .map((c, i) => (c === c.toUpperCase() ? (i !== 0 ? ' ' : '') + c : c))
    .join('')
    .replace(/^./, (str) => str.toUpperCase())
    .replace('Url', 'URL');

export const constructDefaults = (init?: Secret) => {
  if (!init) {
    return {
      name: '',
      type: 'git',
      repoUrl: '',
      repoUrlIsRegex: false,
      username: '',
      password: ''
    };
  }
  return {
    name: init?.metadata?.name || '',
    type: init?.metadata?.labels[CredentialTypeLabelKey] || 'git',
    repoUrl: init?.stringData[CredentialsDataKey.RepoUrl],
    repoUrlIsRegex: init?.stringData[CredentialsDataKey.RepoUrlIsRegex] === 'true',
    username: init?.stringData[CredentialsDataKey.Username],
    password: ''
  };
};
