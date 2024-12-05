export type CredentialsType = 'git' | 'helm' | 'image' | 'generic';
export const CredentialTypeLabelKey = 'kargo.akuity.io/cred-type';

export enum CredentialsDataKey {
  RepoUrl = 'repoURL',
  RepoUrlIsRegex = 'repoURLIsRegex',
  Username = 'username',
  Password = 'password'
}
