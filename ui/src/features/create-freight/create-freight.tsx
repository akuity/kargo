import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { Table } from 'antd';
import { useMemo, useState } from 'react';

import {
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  GitDiscoveryResult,
  Image,
  ImageDiscoveryResult,
  Warehouse
} from '@ui/gen/v1alpha1/generated_pb';

import { FreightContents } from '../freightline/freight-contents';

import { ArtifactMenuGroup } from './artifact-menu-group';
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
  const freight = new Freight();
  freight.warehouse = warehouse;
  freight.images = [];
  freight.charts = [];
  freight.commits = [];

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
      freight.charts.push();
    } else if ('commits' in artifact) {
      freight.commits.push();
    }
  }

  return freight;
};

const ChartTable = ({ versions }: { versions: string[] }) => {
  return (
    <Table
      dataSource={versions.map((version) => ({ version }))}
      columns={[{ title: 'Version', dataIndex: 'version' }]}
    />
  );
};

const GitTable = ({ commits }: { commits: DiscoveredCommit[] }) => {
  return (
    <Table
      dataSource={commits.map((commit) => ({ commit }))}
      columns={[{ title: 'Commit', dataIndex: 'commit' }]}
    />
  );
};

export const CreateFreight = ({ warehouse }: { warehouse?: Warehouse }) => {
  const [selected, setSelected] = useState<DiscoveryResult>();
  const [chosenItems, setChosenItems] = useState<{
    [key: string]: {
      artifact: DiscoveryResult;
      info: FreightInfo;
    };
  }>({});

  // a map of artifact identifiers to freight info
  // contains freight info for all artifacts selected to be included in the new freight
  const [images, charts, git] = useMemo(() => {
    let images: ImageDiscoveryResult[] = [];
    let charts: ChartDiscoveryResult[] = [];
    let git: GitDiscoveryResult[] = [];

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

    if (images?.length > 0) {
      setSelected(images[0]);
    } else if (charts?.length > 0) {
      setSelected(charts[0]);
    } else if (git?.length > 0) {
      setSelected(git[0]);
    }

    return [images, charts, git];
  }, [warehouse]);

  const DiscoveryTable = () => {
    if (!selected) {
      return null;
    }

    if ('references' in selected) {
      return (
        <ImageTable
          references={(selected as ImageDiscoveryResult).references}
          select={(item) => {
            if (item) {
              setChosenItems({
                ...chosenItems,
                [selected.repoURL as string]: {
                  artifact: selected,
                  info: item as DiscoveredImageReference
                }
              });
            } else {
              // eslint-disable-next-line @typescript-eslint/no-unused-vars
              const { [selected.repoURL as string]: _, ...rest } = chosenItems;
              setChosenItems(rest);
            }
          }}
          selected={chosenItems[selected?.repoURL as string]?.info as DiscoveredImageReference}
        />
      );
    } else if ('commits' in selected) {
      return <GitTable commits={(selected as GitDiscoveryResult).commits} />;
    } else if ('versions' in selected) {
      return <ChartTable versions={(selected as ChartDiscoveryResult).versions} />;
    }
  };

  const commonProps = {
    onClick: setSelected,
    selected: selected
  };

  return (
    <div>
      <div className='mb-4 h-12 flex items-center'>
        {Object.keys(chosenItems)?.length > 0 ? (
          <FreightContents
            freight={constructFreight(chosenItems, warehouse?.metadata?.name || '')}
            highlighted
            horizontal
          />
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
