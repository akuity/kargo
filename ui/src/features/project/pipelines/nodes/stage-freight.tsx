import { faChevronLeft, faChevronRight, faCodeCommit } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tag, Typography } from 'antd';
import Link from 'antd/es/typography/Link';
import { ReactNode, useEffect, useMemo, useState } from 'react';

import { getCurrentFreight } from '@ui/features/common/utils';
import {
  getGitCommitURL,
  getImageBuiltDate,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import {
  ArtifactReference,
  Chart,
  FreightReference,
  GitCommit,
  Image,
  Stage
} from '@ui/gen/api/v2/models';

import './stage-node.less';

import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { humanComprehendableArtifact } from '../freight/artifact-parts-utils';
import { shortVersion } from '../freight/short-version-utils';

export const StageFreight = (props: { stage: Stage }) => {
  const dictionaryContext = useDictionaryContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const currentFreight = useMemo(() => getCurrentFreight(props.stage), [props.stage]);

  const warehouses = currentFreight?.map((f) => f.origin?.name);

  const [selectedWarehouse, setSelectedWarehouse] = useState(warehouses?.[0]);

  useEffect(() => {
    if (selectedWarehouse) {
      return;
    }

    setSelectedWarehouse(warehouses?.[0]);
  }, [currentFreight]);

  const defaultToFirstFreight = () =>
    currentFreight?.find((f) => f?.origin?.name === selectedWarehouse) as FreightReference;

  const [selectedFreight, setSelectedFreight] = useState(defaultToFirstFreight);

  useEffect(() => setSelectedFreight(defaultToFirstFreight()), [selectedWarehouse, props.stage]);

  const selectedFreightAlias = useMemo(
    () => dictionaryContext?.freightById?.[selectedFreight?.name || '']?.alias,
    [selectedFreight]
  );

  // Each piece of Freight keeps its artifacts in separate, typed arrays, so the
  // kind of every artifact is known here without inspecting its shape. We render
  // each array with its dedicated component and only then collect the rendered
  // nodes into a single list for the carousel to step through.
  const artifactNodes = useMemo<ReactNode[]>(() => {
    const nodes: ReactNode[] = [];

    for (const image of selectedFreight?.images || []) {
      nodes.push(<ImageArtifact key={`image/${image.repoURL}`} image={image} />);
    }

    for (const commit of selectedFreight?.commits || []) {
      nodes.push(<GitCommitArtifact key={`commit/${commit.repoURL}`} commit={commit} />);
    }

    for (const chart of selectedFreight?.charts || []) {
      nodes.push(<ChartArtifact key={`chart/${chart.repoURL}`} chart={chart} />);
    }

    for (const other of selectedFreight?.artifacts || []) {
      nodes.push(<GenericArtifact key={`generic/${other.subscriptionName}`} artifact={other} />);
    }

    return nodes;
  }, [selectedFreight]);

  const totalArtifacts = artifactNodes.length;

  const [artifactIndex, setArtifactIndex] = useState(0);

  useEffect(() => setArtifactIndex(0), [selectedFreight, props.stage]);

  const onNextArtifact = () => setArtifactIndex((index) => (index + 1) % totalArtifacts);

  const onPreviousArtifact = () =>
    setArtifactIndex((index) => (index - 1 + totalArtifacts) % totalArtifacts);

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
    return null;
  }

  return (
    <>
      {warehouses?.length > 1 && (
        <Flex align='center' justify='center' className='text-[10px] font-light'>
          <Button
            icon={<FontAwesomeIcon icon={faChevronLeft} />}
            size='small'
            type='text'
            className='mr-auto text-[10px]'
            onClick={onPreviousWarehouse}
          />
          {selectedWarehouse}
          <Button
            icon={<FontAwesomeIcon icon={faChevronRight} />}
            size='small'
            type='text'
            className='ml-auto text-[10px]'
            onClick={onNextWarehouse}
          />
        </Flex>
      )}

      <Flex align='center' justify='center' className='my-2'>
        {totalArtifacts > 1 && (
          <Button
            icon={<FontAwesomeIcon icon={faChevronLeft} />}
            size='small'
            type='text'
            className='mr-auto text-[10px]'
            onClick={onPreviousArtifact}
          />
        )}

        <div className='scale-90 flex flex-col items-center min-w-0 overflow-hidden'>
          {freightTimelineControllerContext?.preferredFilter?.showAlias && (
            <div className='text-[10px] mr-1 text-center mb-1'>{selectedFreightAlias}</div>
          )}

          {totalArtifacts === 0 ? (
            <Typography.Text type='secondary' className='text-xs'>
              Empty Freight
            </Typography.Text>
          ) : (
            (artifactNodes[artifactIndex] ?? artifactNodes[0])
          )}

          {freightCreation && (
            <Typography.Text
              type='secondary'
              className='text-[8px] mt-1'
              title={freightCreation?.abs?.toString()}
            >
              {freightCreation?.relative}
            </Typography.Text>
          )}
        </div>

        {totalArtifacts > 1 && (
          <Button
            icon={<FontAwesomeIcon icon={faChevronRight} />}
            size='small'
            type='text'
            className='ml-auto text-[10px]'
            onClick={onNextArtifact}
          />
        )}
      </Flex>
    </>
  );
};

const ArtifactSource = (props: { artifact: GitCommit | Chart | Image }) => (
  <span className='text-[10px] ml-1'>{humanComprehendableArtifact(props.artifact)}</span>
);

const ImageArtifact = (props: { image: Image }) => {
  const { image } = props;

  let imageSourceFromOci = '';
  let imageBuiltDate = '';

  if (image.annotations) {
    imageSourceFromOci = getImageSource(image.annotations);
    imageBuiltDate = getImageBuiltDate(image.annotations);
  }

  let TagComponent = (
    <Tag title={`${image.repoURL}:${image.tag}`} bordered={false} color='geekblue'>
      <Flex justify='center' wrap>
        <div className='text-center'>{shortVersion(image?.tag)}</div>
        <ArtifactSource artifact={image} />
      </Flex>
    </Tag>
  );

  if (imageSourceFromOci) {
    TagComponent = (
      <Link href={imageSourceFromOci} target='_blank' onClick={(e) => e.stopPropagation()}>
        {TagComponent}
      </Link>
    );
  }

  if (imageBuiltDate) {
    TagComponent = (
      <Flex vertical gap={8}>
        {TagComponent}
        <Typography.Text className='text-[10px] text-center' type='secondary'>
          {imageBuiltDate}
        </Typography.Text>
      </Flex>
    );
  }

  return TagComponent;
};

const GitCommitArtifact = (props: { commit: GitCommit }) => {
  const { commit } = props;

  const url = getGitCommitURL(commit.repoURL || '', commit.id || '');

  let TagComponent = (
    <Tag title={commit.repoURL} bordered={false} color='geekblue'>
      <Flex justify='center' align='center' wrap>
        <div>{commit.id?.slice(0, 7)}</div>
        <ArtifactSource artifact={commit} />
      </Flex>
    </Tag>
  );

  if (url) {
    TagComponent = (
      <Link href={url} target='_blank' onClick={(e) => e.stopPropagation()}>
        {TagComponent}
      </Link>
    );
  }

  return (
    <Flex vertical gap={2} align='center'>
      {TagComponent}
      <Typography.Text
        className='text-[10px] text-center'
        type='secondary'
        title={`${commit.author}: ${commit.message}`}
      >
        <FontAwesomeIcon icon={faCodeCommit} className='mr-1' />
        {commit.message?.slice(0, 35)}
        {(commit?.message?.length || 0) > 35 ? '...' : ''}
      </Typography.Text>
    </Flex>
  );
};

const ChartArtifact = (props: { chart: Chart }) => {
  const { chart } = props;

  return (
    <Tag title={`${chart.repoURL}:${chart.version}`} bordered={false} color='geekblue'>
      <Flex wrap justify='center'>
        <div>{shortVersion(chart.version)}</div>
        <ArtifactSource artifact={chart} />
      </Flex>
    </Tag>
  );
};

const GenericArtifact = (props: { artifact: ArtifactReference }) => (
  <Tag bordered={false} color='geekblue'>
    {props.artifact.version}
  </Tag>
);
