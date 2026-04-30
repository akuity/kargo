import { IconDefinition } from '@fortawesome/fontawesome-svg-core';

export type CliBinaries = {
  darwin: { amd64: string; arm64: string };
  linux: { amd64: string; arm64: string };
  windows: { amd64: string; arm64: string };
};

export type Release = {
  version: string;
  latest?: boolean;
  cliBinaries: CliBinaries;
};

export type PlatformLink = {
  title: string;
  getUrl: (release?: Release) => string;
};

export type Platform = {
  title: string;
  icon: IconDefinition;
  links: PlatformLink[];
};
