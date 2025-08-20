import { ConnectError } from '@connectrpc/connect';
import { useMutation } from '@connectrpc/connect-query';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { Button, message, notification, Alert } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useParams, Link, generatePath } from 'react-router-dom';
import yaml from 'yaml';

import { newErrorHandler, newTransportWithAuth } from '@ui/config/transport';
import { createResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
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
} from '@ui/gen/api/v1alpha1/generated_pb';
import { paths } from '@ui/config/paths';

import { FreightContents } from '../freight-timeline/freight-contents';

import { ArtifactMenuGroup } from './artifact-menu-group';
import { ChartTable } from './chart-table';
import { CommitTable } from './commit-table';
import { ImageTable } from './image-table';
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
    }
  }

  return freight;
};

export const AssembleFreight = ({
  warehouse,
  sourceFreight,
  onSuccess
}: {
  warehouse?: Warehouse;
  sourceFreight?: Freight;
  onSuccess: () => void;
}) => {
  const { name: project } = useParams();
  const [selected, setSelected] = useState<DiscoveryResult>();

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

  const [images, charts, git] = useMemo(() => {
    if (!warehouse?.status?.discoveredArtifacts) {
      return [[], [], []];
    }
    return [
      warehouse.status.discoveredArtifacts.images || [],
      warehouse.status.discoveredArtifacts.charts || [],
      warehouse.status.discoveredArtifacts.git || []
    ];
  }, [warehouse]);

  const { init, fallbackUsed, noArtifacts, missing } = useMemo(() => {
    const initialSelections: { [key: string]: { artifact: DiscoveryResult; info: FreightInfo } } =
      {};
    let fallback = false;

    const imagesList = images || [];
    const chartsList = charts || [];
    const gitList = git || [];

    const nothingDiscovered =
      imagesList.length === 0 && chartsList.length === 0 && gitList.length === 0;

    const missingImages: string[] = [];
    const missingCharts: string[] = [];
    const missingCommits: string[] = [];

    const shortCommit = (id?: string) => (id ? id.substring(0, 7) : '');

    if (sourceFreight) {
      for (const image of sourceFreight.images || []) {
        const artifact = imagesList.find((i) => i.repoURL === image.repoURL);
        if (artifact) {
          const info = artifact.references.find(
            (r: DiscoveredImageReference) => r.tag === image.tag
          );
          if (info) {
            initialSelections[getSubscriptionKey(artifact)] = { artifact, info };
          } else {
            missingImages.push(`${image.repoURL}:${image.tag || image.digest || ''}`);
          }
        } else {
          missingImages.push(`${image.repoURL}:${image.tag || image.digest || ''}`);
        }
      }
      for (const chart of sourceFreight.charts || []) {
        const artifact = chartsList.find((c) => c.repoURL === chart.repoURL && c.name === chart.name);
        if (artifact) {
          const info = artifact.versions.find((v: string) => v === chart.version);
          if (info) {
            initialSelections[getSubscriptionKey(artifact)] = { artifact, info };
          } else {
            missingCharts.push(`${chart.name} ${chart.version} (${chart.repoURL})`);
          }
        } else {
          missingCharts.push(`${chart.name} ${chart.version} (${chart.repoURL})`);
        }
      }
      for (const commit of sourceFreight.commits || []) {
        const artifact = gitList.find((g) => g.repoURL === commit.repoURL);
        if (artifact) {
          const info = artifact.commits.find((c: DiscoveredCommit) => c.id === commit.id);
          if (info) {
            initialSelections[getSubscriptionKey(artifact)] = { artifact, info };
          } else {
            missingCommits.push(`${shortCommit(commit.id)} (${commit.repoURL})`);
          }
        } else {
          missingCommits.push(`${shortCommit(commit.id)} (${commit.repoURL})`);
        }
      }

      // Failover to defaults for any discovered artifact with no source match
      for (const image of imagesList) {
        const key = getSubscriptionKey(image);
        if (!initialSelections[key] && image.references?.length > 0) {
          initialSelections[key] = { artifact: image, info: image.references[0] };
          fallback = true;
        }
      }
      for (const chart of chartsList) {
        const key = getSubscriptionKey(chart);
        if (!initialSelections[key] && chart.versions?.length > 0) {
          initialSelections[key] = { artifact: chart, info: chart.versions[0] };
          fallback = true;
        }
      }
      for (const commit of gitList) {
        const key = getSubscriptionKey(commit);
        if (!initialSelections[key] && commit.commits?.length > 0) {
          initialSelections[key] = { artifact: commit, info: commit.commits[0] };
          fallback = true;
        }
      }
    } else {
      for (const image of imagesList) {
        if (image.references?.length > 0) {
          initialSelections[getSubscriptionKey(image)] = {
            artifact: image,
            info: image.references[0]
          };
        }
      }
      for (const chart of chartsList) {
        if (chart.versions?.length > 0) {
          initialSelections[getSubscriptionKey(chart)] = {
            artifact: chart,
            info: chart.versions[0]
          };
        }
      }
      for (const commit of gitList) {
        if (commit.commits?.length > 0) {
          initialSelections[getSubscriptionKey(commit)] = {
            artifact: commit,
            info: commit.commits[0]
          };
        }
      }
    }

    return {
      init: initialSelections,
      fallbackUsed: fallback || (sourceFreight ? nothingDiscovered : false),
      noArtifacts: nothingDiscovered,
      missing: {
        images: missingImages,
        charts: missingCharts,
        commits: missingCommits
      }
    };
  }, [sourceFreight, images, charts, git]);

  const [chosenItems, setChosenItems] = useState<{
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  }>(init);

  useEffect(() => {
    setChosenItems(init);
  }, [init]);

  useEffect(() => {
    const all: DiscoveryResult[] = [
      ...(images as DiscoveryResult[]),
      ...(charts as DiscoveryResult[]),
      ...(git as DiscoveryResult[])
    ];

    const selectedStillExists = selected
      ? all.some((a) => getSubscriptionKey(a) === getSubscriptionKey(selected))
      : false;

    if (!selected || !selectedStillExists) {
      if (images?.length > 0) {
        setSelected(images[0]);
      } else if (charts?.length > 0) {
        setSelected(charts[0]);
      } else if (git?.length > 0) {
        setSelected(git[0]);
      }
    }
  }, [selected, images, charts, git]);

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

  const renderDisclaimer = () => {
    if (!sourceFreight) return null;

    const maxItemsToShow = 3;
    const totalMissing =
      (missing?.images?.length || 0) +
      (missing?.charts?.length || 0) +
      (missing?.commits?.length || 0);

    const missingParts: string[] = [];
    if ((missing?.images?.length || 0) > 0) missingParts.push(`${missing.images.length} image${missing.images.length > 1 ? 's' : ''}`);
    if ((missing?.charts?.length || 0) > 0) missingParts.push(`${missing.charts.length} chart${missing.charts.length > 1 ? 's' : ''}`);
    if ((missing?.commits?.length || 0) > 0) missingParts.push(`${missing.commits.length} commit${missing.commits.length > 1 ? 's' : ''}`);

    const combinedMissing = [
      ...(missing?.images?.map((m) => `image: ${m}`) || []),
      ...(missing?.charts?.map((m) => `chart: ${m}`) || []),
      ...(missing?.commits?.map((m) => `commit: ${m}`) || [])
    ];

    const itemsToShow = combinedMissing.slice(0, maxItemsToShow);
    const remaining = Math.max(0, totalMissing - itemsToShow.length);

    const alertType: 'info' | 'warning' = noArtifacts || totalMissing > 0 ? 'warning' : 'info';

    const isPurePrepopulated = !noArtifacts && totalMissing === 0;

    return (
      <Alert
        type={alertType}
        showIcon
        className='mb-3'
        message={
          <span className='text-base leading-tight'>
            {'Using pre-populated artifacts from '}
            <Link
              to={generatePath(paths.freight, {
                name: project,
                freightName: sourceFreight?.metadata?.name || ''
              })}
            >
              {sourceFreight?.alias || sourceFreight?.metadata?.name}
            </Link>
          </span>
        }
        description={
          <div className='text-sm leading-snug'>
            {isPurePrepopulated && <div>You can adjust them below.</div>}
            {noArtifacts && (
              <div>
                No artifacts are currently discovered for this warehouse. You may need to wait for
                discovery or adjust subscriptions.
              </div>
            )}
            {!noArtifacts && totalMissing > 0 && (
              <div>
                Not found in this warehouse: {missingParts.join(', ')}.
                {fallbackUsed && (
                  <>
                    {' '}Replaced with defaults
                    {itemsToShow.length > 0 && (
                      <div className='mt-1'>
                        Replaced items: {itemsToShow.join(', ')}
                        {remaining > 0 ? `, and ${remaining} more.` : '.'}
                      </div>
                    )}
                  </>
                )}
              </div>
            )}
            {!noArtifacts && totalMissing === 0 && fallbackUsed && (
              <div>Default selections were used. Adjust as needed.</div>
            )}
          </div>
        }
      />
    );
  };

  return (
    <div>
      {renderDisclaimer()}
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
    </>
  );
};
