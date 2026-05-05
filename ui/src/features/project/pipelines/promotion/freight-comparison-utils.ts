import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faArrowDownShortWide } from '@fortawesome/free-solid-svg-icons';
import { formatDistance } from 'date-fns';

import { TableSource } from '@ui/features/freight/flatten-freight-origin-utils';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

export const typeIcon = (source: TableSource): IconProp => {
  switch (source.type) {
    case 'helm':
      return faAnchor;
    case 'image':
      return faDocker;
    case 'git':
      return faGitAlt;
    default:
      return faArrowDownShortWide;
  }
};

export const typeLabel = (source: TableSource): string => {
  switch (source.type) {
    case 'image':
      return 'container image';
    case 'helm':
      return 'helm chart';
    case 'git':
      return source.branch ? `git · branch ${source.branch}` : 'git';
    default:
      return source.artifactType || 'artifact';
  }
};

export const repoLabel = (source: TableSource): string => {
  if (source.type === 'other') {
    return source.subscriptionName || '-';
  }
  return source.repoURL || '-';
};

export const versionLabel = (source: TableSource): string => {
  switch (source.type) {
    case 'git':
      return (source.id || '').slice(0, 8);
    case 'image':
      return source.tag || '';
    case 'helm':
      return source.version || '';
    default:
      return source.version || '';
  }
};

export const relativeFromFreight = (freight: Freight | undefined): string => {
  const created = timestampDate(freight?.metadata?.creationTimestamp);
  if (!created) {
    return '';
  }
  return formatDistance(created, new Date(), { addSuffix: true }).replace('about ', '');
};
