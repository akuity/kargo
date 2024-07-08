import { useMutation } from '@connectrpc/connect-query';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { Button, message } from 'antd';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';

import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import {
  Chart,
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  GitCommit,
  GitDiscoveryResult,
  Image,
  ImageDiscoveryResult,
  Warehouse
} from '@ui/gen/v1alpha1/generated_pb';

import { FreightContents } from '../freight-timeline/freight-contents';

import { ArtifactMenuGroup } from './artifact-menu-group';
import { ChartTable } from './chart-table';
import { CommitTable } from './commit-table';
import { ImageTable } from './image-table';
import { DiscoveryResult, FreightInfo } from './types';

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
    warehouse: warehouse,
    images: [] as Image[],
    charts: [] as Chart[],
    commits: [] as GitCommit[]
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
        gitRepoURL: imageRef.gitRepoURL
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
    }
  }

  return freight;
};

export const CreateFreight = ({
  warehouse,
  onSuccess
}: {
  warehouse?: Warehouse;
  onSuccess: () => void;
}) => {
  const { name: project } = useParams();
  const [selected, setSelected] = useState<DiscoveryResult>();

  const { mutate } = useMutation(createResource, {
    onSuccess: () => {
      message.success('Freight created successfully.');
      onSuccess();
    },
    onError: () => {
      message.error('Failed to create freight.');
    }
  });

  // a map of artifact identifiers to freight info
  // contains freight info for all artifacts selected to be included in the new freight
  const [images, charts, git, init] = useMemo(() => {
    let images: ImageDiscoveryResult[] = [];
    let charts: ChartDiscoveryResult[] = [];
    let git: GitDiscoveryResult[] = [];
    const init = {} as { [key: string]: { artifact: DiscoveryResult; info: FreightInfo } };

    if (!warehouse) {
      return [images, charts, git];
    }

    const discoveredArtifacts = warehouse?.status?.discoveredArtifacts;
    if (!discoveredArtifacts) {
      return [images, charts, git];
    }

    images = discoveredArtifacts.images;
    charts = discoveredArtifacts.charts;
    git = discoveredArtifacts.git;

    if (!selected) {
      if (images?.length > 0) {
        setSelected(images[0]);
      } else if (charts?.length > 0) {
        setSelected(charts[0]);
      } else if (git?.length > 0) {
        setSelected(git[0]);
      }
    }

    for (const image of images) {
      init[image.repoURL as string] = {
        artifact: image,
        info: image.references[0]
      };
    }

    for (const chart of charts) {
      init[chart.repoURL as string] = {
        artifact: chart,
        info: chart.versions[0]
      };
    }

    for (const commit of git) {
      init[commit.repoURL as string] = {
        artifact: commit,
        info: commit.commits[0]
      };
    }

    return [images, charts, git, init];
  }, [warehouse]);

  const [chosenItems, setChosenItems] = useState<{
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  }>(init || {});

  function select<T extends FreightInfo>(item?: T) {
    if (!selected) {
      return;
    }
    if (item) {
      setChosenItems({
        ...chosenItems,
        [selected.repoURL as string]: {
          artifact: selected,
          info: item
        }
      });
    }
  }

  const DiscoveryTable = () => {
    if (!selected) {
      return null;
    }

    const selectedItem = chosenItems[selected?.repoURL as string]?.info;

    if ('references' in selected) {
      return (
        <ImageTable
          references={(selected as ImageDiscoveryResult).references}
          select={select}
          selected={selectedItem as DiscoveredImageReference}
        />
      );
    } else if ('commits' in selected) {
      return (
        <CommitTable
          commits={(selected as GitDiscoveryResult).commits}
          select={select}
          selected={selectedItem as DiscoveredCommit}
        />
      );
    } else if ('versions' in selected) {
      return (
        <ChartTable
          versions={(selected as ChartDiscoveryResult).versions}
          select={select}
          selected={selectedItem as string}
        />
      );
    }
  };

  const commonProps = {
    onClick: setSelected,
    selected: selected
  };

  return (
    <div>
      <div className='text-xs font-medium text-neutral-500 mb-2'>FREIGHT CONTENTS</div>
      <div className='mb-4 h-12 flex items-center'>
        {Object.keys(chosenItems)?.length > 0 ? (
          <>
            <FreightContents
              freight={constructFreight(chosenItems, warehouse?.metadata?.name || '')}
              highlighted
              horizontal
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
          <div className='text-neutral-400'>
            Freight contents will appear here once you select artifacts below.
          </div>
        )}
      </div>
      {warehouse ? (
        <div className='flex w-full border border-solid border-neutral-200 rounded-md overflow-hidden'>
          <div className='bg-neutral-50 p-4' style={{ width: '250px' }}>
            <ArtifactMenuGroup icon={faDocker} label='Images' items={images} {...commonProps} />
            <ArtifactMenuGroup icon={faAnchor} label='Charts' items={charts} {...commonProps} />
            <ArtifactMenuGroup icon={faGit} label='Git' items={git} {...commonProps} />
          </div>
          <div className='w-full p-4 overflow-auto'>
            <DiscoveryTable />
          </div>
        </div>
      ) : (
        <div className='text-neutral-500 text-sm mt-2'>Please select a warehouse to continue.</div>
      )}
    </div>
  );
};
