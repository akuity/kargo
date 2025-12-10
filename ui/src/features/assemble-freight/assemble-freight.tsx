import { ConnectError } from '@connectrpc/connect';
import { useMutation } from '@connectrpc/connect-query';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { Button, message, notification } from 'antd';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';

import { newErrorHandler, newTransportWithAuth } from '@ui/config/transport';
import { WarehouseExpanded } from '@ui/extend/types';
import { createResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import {
  Chart,
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  ArtifactReference,
  DiscoveryResult as GenericDiscoveryResult,
  GitCommit,
  GitDiscoveryResult,
  Image,
  ImageDiscoveryResult
} from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightContents } from '../freight-timeline/freight-contents';

import { ArtifactMenuGroup } from './artifact-menu-group';
import { ChartTable } from './chart-table';
import { CloneFreightNote } from './clone-freight-note';
import { CommitTable } from './commit-table';
import { GenericArtifactTable } from './generic-artifact-table';
import { ImageTable } from './image-table';
import { mergeWithClonedFreight } from './merge-with-cloned-freight';
import { missingArtifactsToClonedFreight } from './missing-artifacts-to-cloned-freight';
import { DiscoveryResult, FreightInfo } from './types';
import { getSubscriptionKey } from './unique-subscription-key';

const constructFreight = (
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
      freight.images.push({
        repoURL: artifact.repoURL,
        tag: imageRef.tag,
        digest: imageRef.digest,
        annotations: imageRef.annotations
      } as Image);
    } else if ('versions' in artifact) {
      freight.charts.push({
        repoURL: artifact.repoURL,
        name: artifact.name,
        version: info as string
      } as Chart);
    } else if ('commits' in artifact) {
      const commitRef = info as DiscoveredCommit;
      freight.commits.push({
        repoURL: artifact.repoURL,
        id: commitRef.id,
        message: commitRef.subject,
        branch: commitRef.branch,
        tag: commitRef.tag,
        author: commitRef.author,
        committer: commitRef.committer
      } as GitCommit);
    } else if ('artifactReferences' in artifact) {
      freight.artifacts.push(info as ArtifactReference);
    }
  }

  return freight;
};

export const AssembleFreight = ({
  warehouse,
  cloneFreight,
  onSuccess
}: {
  warehouse?: WarehouseExpanded;
  cloneFreight?: Freight;
  onSuccess: () => void;
}) => {
  const { name: project } = useParams();

  const errorHandler = newErrorHandler((err) => {
    const errorMessage = err instanceof ConnectError ? err.rawMessage : 'Unexpected API error';
    if (!errorMessage.includes('already exists')) {
      notification.error({ message: errorMessage, placement: 'bottomRight' });
    } else {
      notification.warning({
        message: 'Oops! Freight with these contents already exists.',
        placement: 'bottomRight'
      });
    }
  });

  const { mutate } = useMutation(createResource, {
    transport: newTransportWithAuth(errorHandler),
    onSuccess: () => {
      message.success('Freight created successfully.');
      onSuccess();
    }
  });

  // a map of artifact identifiers to freight info
  // contains freight info for all artifacts selected to be included in the new freight
  const [images, charts, git, other] = useMemo(() => {
    const images: ImageDiscoveryResult[] = warehouse?.status?.discoveredArtifacts?.images || [];
    const charts: ChartDiscoveryResult[] = warehouse?.status?.discoveredArtifacts?.charts || [];
    const git: GitDiscoveryResult[] = warehouse?.status?.discoveredArtifacts?.git || [];
    const other: GenericDiscoveryResult[] = warehouse?.status?.discoveredArtifacts?.results || [];

    return [images, charts, git, other];
  }, [warehouse]);

  const [selected, setSelected] = useState<DiscoveryResult | undefined>(() => {
    if (images?.length > 0) {
      return images[0];
    }

    if (charts?.length > 0) {
      return charts[0];
    }

    if (git?.length > 0) {
      return git[0];
    }

    if (other?.length > 0) {
      return other[0];
    }
  });

  const [chosenItems, setChosenItems] = useState<{
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  }>(() => {
    const items: Record<string, { artifact: DiscoveryResult; info: FreightInfo }> = {};

    const discoveredArtifacts = warehouse?.status?.discoveredArtifacts;

    for (const image of discoveredArtifacts?.images || []) {
      items[getSubscriptionKey(image)] = {
        artifact: image,
        info: image.references[0]
      };
    }

    for (const chart of discoveredArtifacts?.charts || []) {
      items[getSubscriptionKey(chart)] = {
        artifact: chart,
        info: chart.versions[0]
      };
    }

    for (const commit of discoveredArtifacts?.git || []) {
      items[getSubscriptionKey(commit)] = {
        artifact: commit,
        info: commit.commits[0]
      };
    }

    for (const other of discoveredArtifacts?.results || []) {
      items[getSubscriptionKey(other)] = {
        artifact: other,
        info: other.artifactReferences[0]
      };
    }

    if (cloneFreight) {
      mergeWithClonedFreight(items, discoveredArtifacts, cloneFreight);
    }

    return items;
  });

  const missingArtifacts = useMemo(() => {
    if (!cloneFreight) {
      return [];
    }

    return missingArtifactsToClonedFreight(warehouse?.status?.discoveredArtifacts, cloneFreight);
  }, [cloneFreight, warehouse?.status?.discoveredArtifacts]);

  function select<T extends FreightInfo>(item?: T) {
    if (!selected) {
      return;
    }
    if (item) {
      setChosenItems({
        ...chosenItems,
        [getSubscriptionKey(selected)]: {
          artifact: selected,
          info: item
        }
      });
    }
  }

  const commonProps = {
    onClick: setSelected,
    selected: selected
  };

  return (
    <div>
      <CloneFreightNote
        missingArtifacts={missingArtifacts}
        cloneFreight={cloneFreight}
        className='mb-5'
      />
      <div className='text-xs font-medium text-gray-500 mb-2'>FREIGHT CONTENTS</div>
      <div className='mt-3 mb-5 flex items-center'>
        {Object.keys(chosenItems)?.length > 0 ? (
          <>
            <FreightContents
              freight={constructFreight(chosenItems, warehouse?.metadata?.name || '')}
              highlighted
              horizontal
              fullContentVisibility
            />
            <Button
              className='ml-auto'
              onClick={() => {
                const textEncoder = new TextEncoder();
                const freight = constructFreight(chosenItems, warehouse?.metadata?.name || '');

                mutate({
                  manifest: textEncoder.encode(
                    yaml.stringify({
                      kind: 'Freight',
                      apiVersion: 'kargo.akuity.io/v1alpha1',
                      metadata: { name: 'freight', namespace: project },
                      ...freight
                    })
                  )
                });
              }}
            >
              Create
            </Button>
          </>
        ) : (
          <div className='text-gray-400'>
            Freight contents will appear here once you select artifacts below.
          </div>
        )}
      </div>
      {warehouse ? (
        <div className='flex w-full border border-solid border-gray-200 rounded-md overflow-hidden'>
          <div className='bg-gray-50 p-4' style={{ width: '250px' }}>
            <ArtifactMenuGroup icon={faDocker} label='Images' items={images} {...commonProps} />
            <ArtifactMenuGroup icon={faAnchor} label='Charts' items={charts} {...commonProps} />
            <ArtifactMenuGroup icon={faGitAlt} label='Git' items={git} {...commonProps} />
            <ArtifactMenuGroup icon={null} label='Other' items={other} {...commonProps} />
          </div>
          <div className='w-full p-4 overflow-auto'>
            <DiscoveryTable selected={selected} chosenItems={chosenItems} select={select} />
          </div>
        </div>
      ) : (
        <div className='text-gray-500 text-sm mt-2'>Please select a warehouse to continue.</div>
      )}
    </div>
  );
};

const DiscoveryTable = ({
  selected,
  chosenItems,
  select
}: {
  selected?: DiscoveryResult;
  chosenItems: {
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  };
  select: (item?: FreightInfo) => void;
}) => {
  const selectedItem = selected ? chosenItems[getSubscriptionKey(selected)]?.info : undefined;

  if (!selected) {
    return null;
  }

  return (
    <>
      <ImageTable
        references={(selected as ImageDiscoveryResult).references || []}
        select={select}
        selected={selectedItem as DiscoveredImageReference}
        show={'references' in selected}
      />

      <CommitTable
        commits={(selected as GitDiscoveryResult).commits || []}
        select={select}
        selected={selectedItem as DiscoveredCommit}
        show={'commits' in selected}
      />

      <ChartTable
        versions={(selected as ChartDiscoveryResult).versions || []}
        select={select}
        selected={selectedItem as string}
        show={'versions' in selected}
      />

      <GenericArtifactTable
        references={(selected as GenericDiscoveryResult).artifactReferences || []}
        select={select}
        selected={selectedItem as ArtifactReference}
        show={'artifactReferences' in selected}
      />
    </>
  );
};
