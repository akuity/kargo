import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { Button, notification } from 'antd';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';

import { WarehouseExpanded } from '@ui/extend/types';
import {
  ArtifactReference,
  Chart,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';
import { useCreateResource } from '@ui/gen/api/v2/resources/resources';
import { ApiError } from '@ui/lib/api/custom-fetch';

import { FreightContents } from '../freight-timeline/freight-contents';

import { ArtifactMenuGroup } from './artifact-menu-group';
import { ChartTable } from './chart-table';
import { CloneFreightNote } from './clone-freight-note';
import { CommitTable } from './commit-table';
import { GenericArtifactTable } from './generic-artifact-table';
import { ImageTable } from './image-table';
import { mergeWithClonedFreight } from './merge-with-cloned-freight';
import { missingArtifactsToClonedFreight } from './missing-artifacts-to-cloned-freight';
import { ChosenItems, FreightInfo, SelectedArtifact } from './types';
import {
  getArtifactSubscriptionKey,
  getChartSubscriptionKey,
  getGenericSubscriptionKey
} from './unique-subscription-key';

const constructFreight = (chosenItems: ChosenItems, warehouse: string): Freight => {
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

  for (const key in chosenItems.image) {
    const { artifact, info } = chosenItems.image[key];
    freight.images?.push({
      repoURL: artifact.repoURL,
      tag: info.tag,
      digest: info.digest,
      annotations: info.annotations
    } as Image);
  }

  for (const key in chosenItems.chart) {
    const { artifact, info } = chosenItems.chart[key];
    freight.charts?.push({
      repoURL: artifact.repoURL,
      name: artifact.name,
      version: info
    } as Chart);
  }

  for (const key in chosenItems.git) {
    const { artifact, info } = chosenItems.git[key];
    freight.commits?.push({
      repoURL: artifact.repoURL,
      id: info.id,
      message: info.subject,
      branch: info.branch,
      tag: info.tag,
      author: info.author,
      committer: info.committer
    } as GitCommit);
  }

  for (const key in chosenItems.generic) {
    freight.artifacts?.push(chosenItems.generic[key].info);
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

  const { mutate } = useCreateResource({
    mutation: {
      onError: (err) => {
        const errorMessage = err instanceof ApiError ? err.message : 'Unexpected API error';
        if (!errorMessage.includes('already exists')) {
          notification.error({ message: errorMessage, placement: 'bottomRight' });
        } else {
          notification.warning({
            message: 'Oops! Freight with these contents already exists.',
            placement: 'bottomRight'
          });
        }
      },
      onSuccess
    }
  });

  // a map of artifact identifiers to freight info
  // contains freight info for all artifacts selected to be included in the new freight
  const [images, charts, git, other] = useMemo(() => {
    const images = warehouse?.status?.discoveredArtifacts?.images || [];
    const charts = warehouse?.status?.discoveredArtifacts?.charts || [];
    const git = warehouse?.status?.discoveredArtifacts?.git || [];
    const other = warehouse?.status?.discoveredArtifacts?.results || [];

    return [images, charts, git, other];
  }, [warehouse]);

  const [selected, setSelected] = useState<SelectedArtifact | undefined>(() => {
    if (images?.length > 0) {
      return { kind: 'image', value: images[0] };
    }

    if (charts?.length > 0) {
      return { kind: 'chart', value: charts[0] };
    }

    if (git?.length > 0) {
      return { kind: 'git', value: git[0] };
    }

    if (other?.length > 0) {
      return { kind: 'generic', value: other[0] };
    }
  });

  const [chosenItems, setChosenItems] = useState<ChosenItems>(() => {
    const items: ChosenItems = { image: {}, chart: {}, git: {}, generic: {} };

    const discoveredArtifacts = warehouse?.status?.discoveredArtifacts;

    for (const image of discoveredArtifacts?.images || []) {
      items.image[getArtifactSubscriptionKey(image)] = {
        artifact: image,
        info: image.references?.[0] || {}
      };
    }

    for (const chart of discoveredArtifacts?.charts || []) {
      items.chart[getChartSubscriptionKey(chart)] = {
        artifact: chart,
        info: chart.versions?.[0] || ''
      };
    }

    for (const commit of discoveredArtifacts?.git || []) {
      items.git[getArtifactSubscriptionKey(commit)] = {
        artifact: commit,
        info: commit.commits?.[0] || {}
      };
    }

    for (const other of discoveredArtifacts?.results || []) {
      items.generic[getGenericSubscriptionKey(other)] = {
        artifact: other,
        info: other.artifactReferences?.[0] || {}
      };
    }

    if (cloneFreight) {
      mergeWithClonedFreight(items, discoveredArtifacts, cloneFreight);
    }

    return items;
  });

  const missingArtifacts = useMemo(
    () => missingArtifactsToClonedFreight(warehouse?.status?.discoveredArtifacts, cloneFreight),
    [cloneFreight, warehouse?.status?.discoveredArtifacts]
  );

  function select(item?: FreightInfo) {
    if (!selected || !item) {
      return;
    }

    setChosenItems((prev) => {
      switch (selected.kind) {
        case 'image':
          return {
            ...prev,
            image: {
              ...prev.image,
              [getArtifactSubscriptionKey(selected.value)]: {
                artifact: selected.value,
                info: item as DiscoveredImageReference
              }
            }
          };
        case 'chart':
          return {
            ...prev,
            chart: {
              ...prev.chart,
              [getChartSubscriptionKey(selected.value)]: {
                artifact: selected.value,
                info: item as string
              }
            }
          };
        case 'git':
          return {
            ...prev,
            git: {
              ...prev.git,
              [getArtifactSubscriptionKey(selected.value)]: {
                artifact: selected.value,
                info: item as DiscoveredCommit
              }
            }
          };
        case 'generic':
          return {
            ...prev,
            generic: {
              ...prev.generic,
              [getGenericSubscriptionKey(selected.value)]: {
                artifact: selected.value,
                info: item as ArtifactReference
              }
            }
          };
      }
    });
  }

  const totalChosen =
    Object.keys(chosenItems.image).length +
    Object.keys(chosenItems.chart).length +
    Object.keys(chosenItems.git).length +
    Object.keys(chosenItems.generic).length;

  return (
    <div>
      <CloneFreightNote
        missingArtifacts={missingArtifacts}
        cloneFreight={cloneFreight}
        className='mb-5'
      />
      <div className='text-xs font-medium text-gray-500 mb-2'>FREIGHT CONTENTS</div>
      <div className='mt-3 mb-5 flex items-center'>
        {totalChosen > 0 ? (
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
                const freight = constructFreight(chosenItems, warehouse?.metadata?.name || '');

                mutate({
                  data: yaml.stringify({
                    kind: 'Freight',
                    apiVersion: 'kargo.akuity.io/v1alpha1',
                    metadata: { name: 'freight', namespace: project },
                    ...freight
                  })
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
            <ArtifactMenuGroup
              icon={faDocker}
              label='Images'
              items={images}
              getKey={getArtifactSubscriptionKey}
              selected={selected?.kind === 'image' ? selected.value : undefined}
              onClick={(item) => setSelected({ kind: 'image', value: item })}
            />
            <ArtifactMenuGroup
              icon={faAnchor}
              label='Charts'
              items={charts}
              getKey={getChartSubscriptionKey}
              selected={selected?.kind === 'chart' ? selected.value : undefined}
              onClick={(item) => setSelected({ kind: 'chart', value: item })}
            />
            <ArtifactMenuGroup
              icon={faGitAlt}
              label='Git'
              items={git}
              getKey={getArtifactSubscriptionKey}
              selected={selected?.kind === 'git' ? selected.value : undefined}
              onClick={(item) => setSelected({ kind: 'git', value: item })}
            />
            <ArtifactMenuGroup
              icon={null}
              label='Other'
              items={other}
              getKey={getGenericSubscriptionKey}
              selected={selected?.kind === 'generic' ? selected.value : undefined}
              onClick={(item) => setSelected({ kind: 'generic', value: item })}
            />
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
  selected?: SelectedArtifact;
  chosenItems: ChosenItems;
  select: (item?: FreightInfo) => void;
}) => {
  if (!selected) {
    return null;
  }

  switch (selected.kind) {
    case 'image':
      return (
        <ImageTable
          references={selected.value.references || []}
          select={select}
          selected={chosenItems.image[getArtifactSubscriptionKey(selected.value)]?.info}
          show
        />
      );
    case 'git':
      return (
        <CommitTable
          commits={selected.value.commits || []}
          select={select}
          selected={chosenItems.git[getArtifactSubscriptionKey(selected.value)]?.info}
          show
        />
      );
    case 'chart':
      return (
        <ChartTable
          versions={selected.value.versions || []}
          select={select}
          selected={chosenItems.chart[getChartSubscriptionKey(selected.value)]?.info}
          show
        />
      );
    case 'generic':
      return (
        <GenericArtifactTable
          references={selected.value.artifactReferences || []}
          select={select}
          selected={chosenItems.generic[getGenericSubscriptionKey(selected.value)]?.info}
          show
        />
      );
  }
};
