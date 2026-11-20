import {
  ArtifactReference,
  Chart,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';

import { DiscoveryResult, FreightInfo } from './types';

export const constructFreight = (
  chosenItems: {
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  },
  warehouse: string
): Freight => {
  const freight = {
    origin: {
      kind: 'Warehouse',
      name: warehouse
    },
    images: [] as Image[],
    charts: [] as Chart[],
    commits: [] as GitCommit[],
    artifacts: [] as ArtifactReference[]
  } as Freight;

  for (const key in chosenItems) {
    const { artifact, info } = chosenItems[key];
    if ('references' in artifact) {
      const imageRef = info as DiscoveredImageReference;
      if (!imageRef) {
        continue;
      }
      freight.images?.push({
        repoURL: artifact.repoURL,
        tag: imageRef.tag,
        digest: imageRef.digest,
        annotations: imageRef.annotations
      } as Image);
    } else if ('versions' in artifact) {
      freight.charts?.push({
        repoURL: artifact.repoURL,
        name: artifact.name,
        version: info as string,
        annotations: artifact.annotations?.[info as string]
      } as Chart);
    } else if ('commits' in artifact) {
      const commitRef = info as DiscoveredCommit;

      if (!commitRef) {
        continue;
      }

      freight.commits?.push({
        repoURL: artifact.repoURL,
        id: commitRef.id,
        message: commitRef.subject,
        branch: commitRef.branch,
        tag: commitRef.tag,
        author: commitRef.author,
        committer: commitRef.committer
      } as GitCommit);
    } else if ('artifactReferences' in artifact) {
      freight.artifacts?.push(info as ArtifactReference);
    }
  }

  return freight;
};
