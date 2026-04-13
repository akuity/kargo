import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faDharmachakra, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';

import { SegmentLabel } from '@ui/features/common/segment-label';
import {
  DESCRIPTION_ANNOTATION_KEY,
  REPLICATE_TO_ALL_VALUE,
  REPLICATE_TO_ANNOTATION_KEY
} from '@ui/features/common/utils';
import { V1Secret } from '@ui/gen/api/v2/models';

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

export const constructDefaults = (init?: V1Secret, type?: string) => {
  if (!init) {
    return {
      name: '',
      description: '',
      type: type || 'git',
      repoUrl: '',
      repoUrlIsRegex: false,
      username: '',
      password: '',
      data: [],
      replicate: false
    };
  }

  const stringData = init?.stringData ?? {};
  const annotations = init?.metadata?.annotations ?? {};
  const labels = init?.metadata?.labels ?? {};

  return {
    name: init?.metadata?.name || '',
    description: annotations[DESCRIPTION_ANNOTATION_KEY],
    type: labels[CredentialTypeLabelKey] || type || 'git',
    repoUrl: stringData[CredentialsDataKey.RepoUrl],
    repoUrlIsRegex: stringData[CredentialsDataKey.RepoUrlIsRegex] === 'true',
    username: stringData[CredentialsDataKey.Username],
    password: '',
    data: redactSecretStringData(init),
    replicate: annotations[REPLICATE_TO_ANNOTATION_KEY] === REPLICATE_TO_ALL_VALUE
  };
};

export const redactSecretStringData = (secret: V1Secret) => {
  const data = { ...(secret?.stringData ?? {}) };

  for (const key of Object.keys(data)) {
    data[key] = '';
  }

  return Object.entries(data);
};
