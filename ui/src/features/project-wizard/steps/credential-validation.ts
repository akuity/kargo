import { dnsRegex } from '@ui/features/common/utils';

import { CredentialData } from '../types';

const imageNameRegex =
  /^(?![a-zA-Z][a-zA-Z0-9+.-]*:\/\/)(\w+([.-]\w+)*(:\d+)?\/)?(\w+([.-]\w+)*)(\/\w+([.-]\w+)*)*$/;

// Mirrors the per-type refinements in the settings modal's schema-validator
export const repoUrlError = (cred: CredentialData): string | undefined => {
  if (cred.repoURLIsRegex || !cred.repoURL) {
    return undefined;
  }
  switch (cred.type) {
    case 'git':
      try {
        new URL(cred.repoURL);
      } catch {
        return 'Repo URL must be a valid git URL.';
      }
      return undefined;
    case 'helm':
      try {
        const url = new URL(cred.repoURL);
        if (url.protocol !== 'http:' && url.protocol !== 'https:' && url.protocol !== 'oci:') {
          return 'Repo URL must be a valid Helm chart repository.';
        }
      } catch {
        return 'Repo URL must be a valid Helm chart repository.';
      }
      return undefined;
    case 'image':
      return imageNameRegex.test(cred.repoURL)
        ? undefined
        : 'Repo URL must be a valid container registry.';
  }
};

// Required fields per auth method, mirroring the settings modal's refinements
// (repoUrl/username/password required) and pkg/credentials' expectations for
// the advanced shapes.
const hasRequiredAuthFields = (cred: CredentialData): boolean => {
  switch (cred.auth) {
    case 'userpass':
      return !!cred.username && !!cred.password;
    case 'ssh':
      return !!cred.sshPrivateKey;
    case 'github-app':
      return (
        (!!cred.githubAppClientID || !!cred.githubAppID) &&
        !!cred.githubAppInstallationID &&
        !!cred.githubAppPrivateKey
      );
    case 'aws-ecr':
      return !!cred.awsRegion && !!cred.awsAccessKeyID && !!cred.awsSecretAccessKey;
    case 'gcp-ar':
      return !!cred.gcpServiceAccountKey;
    case 'ambient':
      return true;
  }
};

export const isValidCredential = (cred: CredentialData) =>
  dnsRegex.test(cred.name) && !!cred.repoURL && !repoUrlError(cred) && hasRequiredAuthFields(cred);
