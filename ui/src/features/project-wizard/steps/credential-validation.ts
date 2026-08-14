import { repoUrlError as repoUrlShapeError } from '@ui/features/common/settings/secrets/schema-validator';
import { dnsRegex } from '@ui/features/common/utils';

import { CredentialData } from '../types';

// The same per-type repo URL rules the settings modal's schema enforces, in
// terms of the wizard's field names.
export const repoUrlError = (cred: CredentialData): string | undefined =>
  repoUrlShapeError(cred.type, cred.repoURL, cred.repoURLIsRegex);

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
