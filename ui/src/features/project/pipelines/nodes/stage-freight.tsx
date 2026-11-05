import { faChevronLeft, faChevronRight, faPause } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tooltip } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import {
  isArtifactChart,
  isArtifactGeneric,
  isArtifactGitCommit,
  isArtifactImage
} from '@ui/features/assemble-freight/artifact-type-guards';
import { PromotionStatusPhase } from '@ui/features/common/promotion-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import {
  getGitCommitURL,
  getImageBuiltDate,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import { ArtifactReference, Chart, GitCommit, Image, Stage } from '@ui/gen/api/v2/models';

import './stage-node.less';

import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { ArtifactIcon } from '../freight/artifact-icon';
import { humanComprehendableArtifact } from '../freight/artifact-parts-utils';
import { shortVersion } from '../freight/short-version-utils';
import { getAutoPromotionHold, holdStateMessage } from '../promotion/auto-promotion';

import {
  ArtifactTypes,
  resolveArtifact,
  resolveWarehouse,
  selectNextArtifact,
  selectPreviousArtifact
} from './artifact-selector-utils';
import { getLastPromotionRef } from './stage-meta-utils';

export const StageFreight = (props: { stage: Stage }) => {
  const dictionaryContext = useDictionaryContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  // The "All Warehouses" filter governs what the graph draws, so the Freight
  // a node shows has to answer to it too. An empty selection means all.
  const warehouseFilter = freightTimelineControllerContext?.preferredFilter?.warehouses;

  const allFreight = useMemo(() => getCurrentFreight(props.stage), [props.stage]);

  const currentFreight = useMemo(() => {
    if (!warehouseFilter?.length) {
      return allFreight;
    }

    return allFreight?.filter((f) => warehouseFilter.includes(f?.origin?.name || ''));
  }, [allFreight, warehouseFilter]);

  const warehouses = currentFreight?.map((f) => f.origin?.name);

  const lastPromotion = getLastPromotionRef(props.stage);
  const lastSucceededOrigin =
    lastPromotion?.phase === PromotionStatusPhase.SUCCEEDED
      ? lastPromotion.freightOrigin
      : undefined;

  const [preferredWarehouse, setPreferredWarehouse] = useState(lastSucceededOrigin);
  const [preferredArtifact, setPreferredArtifact] = useState<ArtifactTypes>();

  useEffect(() => {
    if (!lastSucceededOrigin) {
      return;
    }

    setPreferredWarehouse(lastSucceededOrigin);
    setPreferredArtifact(undefined);
  }, [lastPromotion?.kind, lastPromotion?.name, lastSucceededOrigin]);

  const selectedWarehouse = resolveWarehouse(warehouses, preferredWarehouse);

  const selectedFreight = useMemo(
    () => currentFreight?.find((f) => f?.origin?.name === selectedWarehouse),
    [currentFreight, selectedWarehouse]
  );

  const setSelectedWarehouse = (warehouse?: string) => {
    setPreferredWarehouse(warehouse);
    setPreferredArtifact(undefined);
  };

  const selectedAutoPromotionHold = useMemo(
    () => getAutoPromotionHold(props.stage, selectedFreight?.origin),
    [props.stage, selectedFreight]
  );

  // The dictionary is a dependency as much as the freight is: it arrives a
  // moment after first paint, and without it here the memo keeps the empty
  // answer and freightLabel falls back to the head of the name for good.
  const selectedFreightAlias = useMemo(
    () => dictionaryContext?.freightById?.[selectedFreight?.name || '']?.alias,
    [dictionaryContext?.freightById, selectedFreight]
  );
  const showFreightAlias = freightTimelineControllerContext?.preferredFilter?.showAlias;

  const selectedArtifact = resolveArtifact(selectedFreight, preferredArtifact);

  const onNextArtifact = () => {
    if (selectedFreight) {
      setPreferredArtifact(selectNextArtifact(selectedFreight, selectedArtifact as ArtifactTypes));
    }
  };

  const onPreviousArtifact = () => {
    if (selectedFreight) {
      setPreferredArtifact(
        selectPreviousArtifact(selectedFreight, selectedArtifact as ArtifactTypes)
      );
    }
  };

  const onNextWarehouse = () => {
    const currentWarehouseIndex = warehouses.findIndex((w) => w === selectedWarehouse);

    const nextWarehouseIndex = (currentWarehouseIndex + 1) % warehouses?.length;

    setSelectedWarehouse(warehouses[nextWarehouseIndex]);
  };

  const onPreviousWarehouse = () => {
    const currentWarehouseIndex = warehouses.findIndex((w) => w === selectedWarehouse);

    let nextWarehouseIndex = currentWarehouseIndex - 1;

    if (nextWarehouseIndex < 0) {
      nextWarehouseIndex = warehouses.length - 1;
    }

    setSelectedWarehouse(warehouses[nextWarehouseIndex]);
  };

  const freightCreation = useGetFreightCreation(
    dictionaryContext?.freightById?.[selectedFreight?.name || '']
  );

  if (!currentFreight?.length) {
    // The absence is news, and the well says so rather than vanishing -- a
    // node that quietly loses a section reads as a rendering fault.
    //
    // The Stage may also hold Freight that the "All Warehouses" filter is
    // keeping out of view. Telling the reader it has none would be a lie, so
    // the two absences are named apart.
    return (
      <div className='stage-node__freight min-w-0'>
        <div className='stage-node__section-label'>Current Freight</div>
        <div className='stage-node__freight-empty'>
          {allFreight?.length ? 'No freight from selected warehouses' : 'No freight yet'}
        </div>
      </div>
    );
  }

  const noOfGitCommits = selectedFreight?.commits?.length || 0;
  const noOfHelmReleases = selectedFreight?.charts?.length || 0;
  const noOfContainerImages = selectedFreight?.images?.length || 0;
  const noOfGenericArtifacts = selectedFreight?.artifacts?.length || 0;

  const totalArtifacts =
    noOfContainerImages + noOfGitCommits + noOfHelmReleases + noOfGenericArtifacts;

  // The alias is what a user recognizes a piece of freight by. With aliases
  // turned off, the head of the generated name is the next best handle.
  const freightLabel =
    (showFreightAlias && selectedFreightAlias) || selectedFreight?.name?.slice(0, 7) || '';

  return (
    <Flex vertical gap={5} className='stage-node__freight min-w-0'>
      <Flex align='center' justify='space-between' gap={8} className='min-w-0'>
        <span className='stage-node__section-label'>Current Freight</span>

        {warehouses?.length > 1 && (
          <Pager
            label={selectedWarehouse}
            onPrevious={onPreviousWarehouse}
            onNext={onNextWarehouse}
          />
        )}
      </Flex>

      <Flex align='center' justify='space-between' gap={8} className='min-w-0'>
        <Flex align='center' gap={4} className='min-w-0'>
          {selectedAutoPromotionHold && (
            <Tooltip title={holdStateMessage(props.stage, selectedFreight?.origin)}>
              <span className='inline-flex text-[10px] text-orange-600'>
                <FontAwesomeIcon icon={faPause} />
              </span>
            </Tooltip>
          )}

          <span className='stage-node__freight-alias' title={freightLabel}>
            {freightLabel}
          </span>
        </Flex>

        {!!freightCreation?.relative && (
          <span className='stage-node__freight-age' title={freightCreation?.abs?.toString()}>
            {freightCreation.relative} old
          </span>
        )}
      </Flex>

      <Flex align='center' gap={2} className='min-w-0'>
        {totalArtifacts > 1 && (
          <Button
            className='stage-node__pager-button'
            icon={<FontAwesomeIcon icon={faChevronLeft} />}
            size='small'
            type='text'
            title='Previous artifact'
            onClick={onPreviousArtifact}
          />
        )}

        <Artifact artifact={selectedArtifact} />

        {totalArtifacts > 1 && (
          <Button
            className='stage-node__pager-button'
            icon={<FontAwesomeIcon icon={faChevronRight} />}
            size='small'
            type='text'
            title='Next artifact'
            onClick={onNextArtifact}
          />
        )}
      </Flex>
    </Flex>
  );
};

// The pager names the Warehouse rather than counting it. The position in the
// list would be an artifact of how the Freight history serializes, and the
// name is the only part a reader can act on.
const Pager = (props: { label?: string; onPrevious: () => void; onNext: () => void }) => (
  <span className='stage-node__pager'>
    <Button
      className='stage-node__pager-button'
      icon={<FontAwesomeIcon icon={faChevronLeft} />}
      size='small'
      type='text'
      title='Previous warehouse'
      onClick={props.onPrevious}
    />

    <span className='stage-node__pager-label' title={props.label}>
      {props.label}
    </span>

    <Button
      className='stage-node__pager-button'
      icon={<FontAwesomeIcon icon={faChevronRight} />}
      size='small'
      type='text'
      title='Next warehouse'
      onClick={props.onNext}
    />
  </span>
);

const subscriptionTitle = (base: string | undefined, subscriptionName: string | undefined) =>
  subscriptionName ? `${base} (subscription: ${subscriptionName})` : base;

// A tooltip gathers everything the tile has no room for -- the full repository,
// the commit message, when an image was built -- one fact per line.
const tileTitle = (...lines: (string | undefined)[]) => lines.filter(Boolean).join('\n');

const ArtifactTile = (props: {
  artifact: GitCommit | Chart | Image | ArtifactReference;
  name?: string;
  version?: string;
  title?: string;
  href?: string;
}) => {
  const tile = (
    <div className='stage-node__artifact' title={props.title}>
      <ArtifactIcon artifact={props.artifact} className='stage-node__artifact-icon' />
      <span className='stage-node__artifact-name'>{props.name}</span>
      <span className='stage-node__artifact-version'>{props.version}</span>
    </div>
  );

  if (!props.href) {
    return tile;
  }

  return (
    <a
      className='stage-node__artifact-link'
      href={props.href}
      target='_blank'
      rel='noreferrer'
      onClick={(e) => e.stopPropagation()}
    >
      {tile}
    </a>
  );
};

const Artifact = (props: { artifact: string | GitCommit | Chart | Image | ArtifactReference }) => {
  if (typeof props.artifact === 'string') {
    return <div className='stage-node__artifact stage-node__artifact--empty'>Empty Freight</div>;
  }

  const artifact = props.artifact;

  // A generic artifact has no repository to name it, so its subscription name
  // is all the tile can show alongside the version.
  if (isArtifactGeneric(artifact)) {
    return (
      <ArtifactTile
        artifact={artifact}
        name={artifact.subscriptionName || 'Artifact'}
        version={shortVersion(artifact.version)}
        title={subscriptionTitle(artifact.version, artifact.subscriptionName)}
      />
    );
  }

  if (isArtifactGitCommit(artifact)) {
    return (
      <ArtifactTile
        artifact={artifact}
        name={humanComprehendableArtifact(artifact)}
        version={artifact.tag ? shortVersion(artifact.tag) : artifact.id?.slice(0, 7)}
        title={tileTitle(
          subscriptionTitle(artifact.repoURL, artifact.subscriptionName),
          artifact.message && `${artifact.author}: ${artifact.message}`
        )}
        href={getGitCommitURL(artifact.repoURL || '', artifact.id || '')}
      />
    );
  }

  if (isArtifactChart(artifact)) {
    return (
      <ArtifactTile
        artifact={artifact}
        name={humanComprehendableArtifact(artifact)}
        version={shortVersion(artifact.version)}
        title={subscriptionTitle(
          `${artifact.repoURL}:${artifact.version}`,
          artifact.subscriptionName
        )}
      />
    );
  }

  if (isArtifactImage(artifact)) {
    const imageSourceFromOci = artifact.annotations ? getImageSource(artifact.annotations) : '';
    const imageBuiltDate = artifact.annotations ? getImageBuiltDate(artifact.annotations) : '';

    return (
      <ArtifactTile
        artifact={artifact}
        name={humanComprehendableArtifact(artifact)}
        version={shortVersion(artifact.tag)}
        title={tileTitle(
          subscriptionTitle(`${artifact.repoURL}:${artifact.tag}`, artifact.subscriptionName),
          imageBuiltDate && `Built ${imageBuiltDate}`
        )}
        href={imageSourceFromOci}
      />
    );
  }
};
