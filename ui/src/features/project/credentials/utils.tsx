import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faDharmachakra, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';

import { SegmentLabel } from '@ui/features/common/segment-label';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

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
    case 'generic':
      return faDharmachakra;
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

export const constructDefaults = (init?: Secret, type?: string) => {
  if (!init) {
    return {
      name: '',
      description: '',
      type: type || 'git',
      repoUrl: '',
      repoUrlIsRegex: false,
      username: '',
      password: '',
      data: []
    };
  }

  return {
    name: init?.metadata?.name || '',
    description: init?.metadata?.annotations[DESCRIPTION_ANNOTATION_KEY],
    type: init?.metadata?.labels[CredentialTypeLabelKey] || type || 'git',
    repoUrl: init?.stringData[CredentialsDataKey.RepoUrl],
    repoUrlIsRegex: init?.stringData[CredentialsDataKey.RepoUrlIsRegex] === 'true',
    username: init?.stringData[CredentialsDataKey.Username],
    password: '',
    data: redactSecretStringData(init)
  };
};

export const redactSecretStringData = (secret: Secret) => {
  const data = secret?.stringData;

  for (const key of Object.keys(data)) {
    data[key] = '';
  }

  return Object.entries(data);
};
