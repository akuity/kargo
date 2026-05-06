import {
  flattenFreightOrigin,
  TableSource
} from '@ui/features/freight/flatten-freight-origin-utils';
import { Freight, FreightReference } from '@ui/gen/api/v1alpha1/generated_pb';

export type PairStatus = 'CHANGED' | 'UNCHANGED' | 'NEW' | 'REMOVED';

export type PairedRow = {
  key: string;
  status: PairStatus;
  current?: TableSource;
  incoming?: TableSource;
};

const pairKey = (s: TableSource): string => {
  if (s.type === 'other') {
    return `other:${s.subscriptionName || ''}`;
  }
  return `${s.type}:${s.repoURL || ''}`;
};

const versionOf = (s: TableSource): string => {
  switch (s.type) {
    case 'git':
      return s.id || '';
    case 'image':
      return s.tag || '';
    case 'helm':
      return s.version || '';
    case 'other':
      return s.version || '';
  }
};

export const pairArtifacts = (
  current: Freight | FreightReference | undefined | null,
  incoming: Freight | FreightReference | undefined | null
): PairedRow[] => {
  const currentItems = flattenFreightOrigin(current);
  const incomingItems = flattenFreightOrigin(incoming);

  const currentByKey = new Map<string, TableSource>();
  for (const item of currentItems) {
    currentByKey.set(pairKey(item), item);
  }

  const rows: PairedRow[] = [];
  const seen = new Set<string>();

  for (const item of incomingItems) {
    const key = pairKey(item);
    seen.add(key);
    const cur = currentByKey.get(key);
    if (!cur) {
      rows.push({ key, status: 'NEW', incoming: item });
    } else if (versionOf(cur) === versionOf(item)) {
      rows.push({ key, status: 'UNCHANGED', current: cur, incoming: item });
    } else {
      rows.push({ key, status: 'CHANGED', current: cur, incoming: item });
    }
  }

  for (const item of currentItems) {
    const key = pairKey(item);
    if (!seen.has(key)) {
      rows.push({ key, status: 'REMOVED', current: item });
    }
  }

  return rows;
};
