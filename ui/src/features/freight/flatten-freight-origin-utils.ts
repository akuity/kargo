import { ArtifactReference, Freight, FreightReference } from '@ui/gen/api/v2/models';

export type TableSource =
  | {
      type: 'image';
      repoURL: string;
      tag?: string;
      annotations?: Record<string, string>;
      subscriptionName?: string;
    }
  | {
      type: 'git';
      repoURL: string;
      id: string;
      branch: string;
      message: string;
      author: string;
      committer: string;
      tag?: string;
      subscriptionName?: string;
    }
  | {
      type: 'helm';
      repoURL: string;
      name: string;
      version: string;
      subscriptionName?: string;
    }
  | ({
      type: 'other';
    } & ArtifactReference);

export const flattenFreightOrigin = (
  freight: Freight | FreightReference | undefined | null
): TableSource[] => {
  const images: TableSource[] =
    freight?.images?.map((image) => ({
      type: 'image',
      repoURL: image?.repoURL || '',
      tag: image?.tag || '',
      annotations: image?.annotations || {},
      subscriptionName: image?.subscriptionName
    })) || [];

  const git: TableSource[] =
    freight?.commits?.map((commit) => ({
      type: 'git',
      repoURL: commit?.repoURL || '',
      author: commit?.author || '',
      branch: commit?.branch || '',
      committer: commit?.committer || '',
      id: commit?.id || '',
      message: commit?.message || '',
      tag: commit?.tag || '',
      subscriptionName: commit?.subscriptionName
    })) || [];

  const helm: TableSource[] =
    freight?.charts?.map((chart) => ({
      type: 'helm',
      repoURL: chart?.repoURL || '',
      name: chart?.name || '',
      version: chart?.version || '',
      subscriptionName: chart?.subscriptionName
    })) || [];

  const other: TableSource[] =
    freight?.artifacts?.map((otherArtifact) => ({
      type: 'other',
      ...otherArtifact
    })) || [];

  return [...images, ...git, ...helm, ...other];
};
