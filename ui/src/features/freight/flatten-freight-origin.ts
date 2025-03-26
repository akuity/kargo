import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

type TableSource =
  | {
      type: 'image';
      repoURL: string;
      tag?: string;
      annotations?: Record<string, string>;
    }
  | {
      type: 'git';
      repoURL: string;
      id: string;
      branch: string;
      message: string;
      author: string;
      committer: string;
    }
  | {
      type: 'helm';
      repoURL: string;
      version: string;
    };

export const flattenFreightOrigin = (freight: Freight): TableSource[] => {
  const images: TableSource[] =
    freight?.images?.map((image) => ({
      type: 'image',
      repoURL: image?.repoURL,
      tag: image?.tag,
      annotations: image?.annotations
    })) || [];

  const git: TableSource[] = freight?.commits?.map((commit) => ({
    type: 'git',
    repoURL: commit?.repoURL,
    author: commit?.author,
    branch: commit?.branch,
    committer: commit?.committer,
    id: commit?.id,
    message: commit?.message
  }));

  const helm: TableSource[] = freight?.charts?.map((chart) => ({
    type: 'helm',
    repoURL: chart?.repoURL,
    version: chart?.version
  }));

  return [...images, ...git, ...helm];
};
