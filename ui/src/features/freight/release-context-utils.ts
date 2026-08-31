import { Freight, FreightReference, Image, ImageAnnotationMappings } from '@ui/gen/api/v2/models';

import {
  getImageSource,
  ociAnnotationKeys
} from '../freight-timeline/open-container-initiative-utils';
import { pairArtifacts, PairStatus } from '../project/pipelines/promotion/pair-artifacts';

export type ImageReleaseContext = {
  image: Image;
  source?: string;
  sourceURL?: string;
  commitURL?: string;
  revision?: string;
  createdAt?: string;
  subject?: string;
  author?: string;
  committer?: string;
  commitCreatedAt?: string;
  annotations: { key: string; value: string }[];
};

export type PairedImageReleaseContext = {
  key: string;
  status: PairStatus;
  current?: ImageReleaseContext;
  incoming?: ImageReleaseContext;
};

const getWebURL = (value?: string): string | undefined => {
  if (!value) {
    return undefined;
  }

  try {
    const url = new URL(value);
    return url.protocol === 'https:' || url.protocol === 'http:' ? url.href : undefined;
  } catch {
    return undefined;
  }
};

const getMappedAnnotation = (
  annotations: Record<string, string>,
  key?: string
): string | undefined =>
  key && !key.startsWith('org.opencontainers.') && Object.hasOwn(annotations, key)
    ? annotations[key] || undefined
    : undefined;

export const getImageReleaseContext = (
  image: Image,
  mappings?: ImageAnnotationMappings
): ImageReleaseContext => {
  const annotations = image.annotations || {};
  const source = annotations[ociAnnotationKeys.source];

  return {
    image,
    source,
    sourceURL: getWebURL(source),
    commitURL: getWebURL(getImageSource(annotations)),
    revision: annotations[ociAnnotationKeys.revision],
    createdAt: annotations[ociAnnotationKeys.createdAt],
    subject: getMappedAnnotation(annotations, mappings?.commitSubject),
    author: getMappedAnnotation(annotations, mappings?.commitAuthor),
    committer: getMappedAnnotation(annotations, mappings?.commitCommitter),
    commitCreatedAt: getMappedAnnotation(annotations, mappings?.commitCreatedAt),
    annotations: Object.entries(annotations)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, value]) => ({ key, value }))
  };
};

export const getFreightImageReleaseContexts = (
  freight: Freight | FreightReference | undefined,
  mappings?: ImageAnnotationMappings
): ImageReleaseContext[] =>
  (freight?.images || []).map((image) => getImageReleaseContext(image, mappings));

export const pairImageReleaseContexts = (
  currentFreight: Freight | FreightReference | undefined,
  incomingFreight: Freight | FreightReference,
  mappings?: ImageAnnotationMappings
): PairedImageReleaseContext[] =>
  pairArtifacts(currentFreight, incomingFreight)
    .filter((pair) => pair.current?.type === 'image' || pair.incoming?.type === 'image')
    .map((pair) => ({
      key: pair.key,
      status: pair.status,
      current:
        pair.current?.type === 'image' ? getImageReleaseContext(pair.current, mappings) : undefined,
      incoming:
        pair.incoming?.type === 'image'
          ? getImageReleaseContext(pair.incoming, mappings)
          : undefined
    }));
