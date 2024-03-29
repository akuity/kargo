import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { Secret } from '@ui/gen/v1alpha1/types_pb';

import { CredentialTypeLabelKey, CredentialsType } from './types';

export const typeLabel = (type: CredentialsType) => (
  <span className='flex items-center font-semibold justify-center text-center p-4'>
    <FontAwesomeIcon icon={iconForCredentialsType(type)} className='mr-2' />
    {type.toUpperCase()}
  </span>
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
      repoUrlPattern: '',
      username: '',
      password: ''
    };
  }
  return {
    name: init?.metadata?.name || '',
    type: init?.metadata?.labels[CredentialTypeLabelKey] || 'git',
    repoUrl: init?.stringData['repoURL'],
    repoUrlPattern: init?.stringData['repoURLPattern'],
    username: init?.stringData['username'],
    password: ''
  };
};
