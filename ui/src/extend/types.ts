// extended types for protobufs
// some protobufs have generic types ie. JSON but we know the exact types

import { Warehouse } from '@ui/gen/api/v2/models';

export type GitSubscription = {
  repoURL: string;
  branch?: string;
  commitSelectionStrategy?: string;
  semverConstraint?: string;
  allowTagsRegexes?: string[];
  ignoreTagsRegexes?: string[];
  includePaths?: string[];
  excludePaths?: string[];
  expressionFilter?: string;
  discoveryLimit?: number;
  insecureSkipTLSVerify?: boolean;
  strictSemvers?: boolean;
  since?: string;
};

export type ImageSubscription = {
  repoURL: string;
  imageSelectionStrategy?: string;
  constraint?: string;
  allowTagsRegexes?: string[];
  ignoreTagsRegexes?: string[];
  platform?: string;
  discoveryLimit?: number;
  insecureSkipTLSVerify?: boolean;
  strictSemvers?: boolean;
  cacheByTag?: boolean;
};

export type ChartSubscription = {
  repoURL: string;
  name?: string;
  semverConstraint?: string;
  discoveryLimit?: number;
  insecureSkipTLSVerify?: boolean;
};

export type Subscription = {
  subscriptionType: string;
  name: string;
  config?: unknown;
  discoveryLimit?: number;
};

export type RepoSubscription = {
  git?: GitSubscription;
  image?: ImageSubscription;
  chart?: ChartSubscription;
  subscription?: Subscription;
};

export type WarehouseExpanded = Omit<Warehouse, 'spec'> & {
  spec?: Omit<Warehouse['spec'], 'subscriptions'> & {
    subscriptions?: RepoSubscription[];
  };
};
